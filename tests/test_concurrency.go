package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	fmt.Println("=================================================================")
	fmt.Println("   PENGUJIAN BATAS 20 PROYEK DENGAN 10 REQUEST SERENTAK (RACE CONDITION)   ")
	fmt.Println("=================================================================")

	// 1. Reset projects di MongoDB menjadi tepat 18 proyek
	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		fmt.Printf("Gagal koneksi Mongo: %v\n", err)
		return
	}
	defer client.Disconnect(context.Background())

	coll := client.Database("mahasiswa_portfolio").Collection("student_portfolios")

	var initialProjects []bson.M
	for i := 1; i <= 18; i++ {
		initialProjects = append(initialProjects, bson.M{
			"_id":          bson.NewObjectID(),
			"title":        fmt.Sprintf("Proyek Otomatis #%02d", i),
			"technologies": []string{"Go", "PostgreSQL"},
			"link":         "https://github.com/farril",
			"year":         2026,
		})
	}

	_, err = coll.UpdateOne(context.Background(),
		bson.M{"student_id": 7},
		bson.M{"$set": bson.M{"projects": initialProjects}},
	)
	if err != nil {
		fmt.Printf("Gagal menyiapkan 18 proyek awal: %v\n", err)
		return
	}
	fmt.Println("[STEP 1] Berhasil menyiapkan tepat 18 proyek awal di portofolio mahasiswa ID 7.")

	// 2. Tembak 10 request serentak secara bersamaan menggunakan Goroutines
	fmt.Println("[STEP 2] Mengirim 10 request POST secara serentak (bersamaan di milidetik yang sama)...")

	var wg sync.WaitGroup
	results := make([]int, 10)
	targetURL := "http://localhost:3000/api/v1/students/7/portfolio/projects"

	startGate := make(chan struct{}) // Gerbang peluru agar 10 request meluncur di waktu yang sama persis

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			payload, _ := json.Marshal(map[string]any{
				"title":        fmt.Sprintf("Proyek Serentak #%d", idx+1),
				"technologies": []string{"Docker", "Kubernetes"},
				"link":         "https://github.com/farril/concurrent",
				"year":         2026,
			})

			<-startGate // Tunggu aba-aba tembak bersamaan

			req, _ := http.NewRequest(http.MethodPost, targetURL, bytes.NewBuffer(payload))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				results[idx] = 500
				return
			}
			results[idx] = resp.StatusCode
			resp.Body.Close()
		}(i)
	}

	time.Sleep(200 * time.Millisecond)
	close(startGate) // Buka gerbang peluru serentak!
	wg.Wait()

	// 3. Hitung hasil
	successCount := 0
	rejectedCount := 0
	for _, status := range results {
		if status == 201 {
			successCount++
		} else if status == 400 || status == 422 {
			rejectedCount++
		}
	}

	// 4. Verifikasi jumlah akhir di database
	var finalDoc struct {
		Projects []bson.M `bson:"projects"`
	}
	_ = coll.FindOne(context.Background(), bson.M{"student_id": 7}).Decode(&finalDoc)

	fmt.Println("-----------------------------------------------------------------")
	fmt.Printf("HASIL PENGUJIAN CONCURRENCY:\n")
	fmt.Printf("• Total request dikirim serentak : 10 request\n")
	fmt.Printf("• Request yang BERHASIL (201)    : %d (Sesuai ekspektasi: tepat 2)\n", successCount)
	fmt.Printf("• Request yang DITOLAK (400)     : %d (Sesuai ekspektasi: 8 ditolak)\n", rejectedCount)
	fmt.Printf("• Total proyek di database akhir : %d proyek (Maksimal 20)\n", len(finalDoc.Projects))
	fmt.Println("-----------------------------------------------------------------")
	if successCount == 2 && len(finalDoc.Projects) == 20 {
		fmt.Println("STATUS: [SUKSES TERBUKTI] Batas 20 proyek terbukti bertahan dari race condition!")
	} else {
		fmt.Println("STATUS: PERLU PENYESUAIAN")
	}
	fmt.Println("=================================================================")
}

//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"api-students/app/model"
	"api-students/config"
	"api-students/database"
)

const (
	mhsToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6Im1oc3VqaSIsInJvbGUiOiJtYWhhc2lzd2EiLCJpc3MiOiJwcmFrdGlrdW0tYmFja2VuZCIsInN1YiI6IjciLCJleHAiOjE3OTMzNzg5MTgsImlhdCI6MTc5MzI5MjUxOH0.N7l2RzRLRggy3OCVuvoOWguumiYmggUjx2_H1Yt0IQw"
	baseURL  = "http://localhost:3000"
)

func doReq(cid int) int {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/courses/%d/enroll", baseURL, cid), nil)
	if err != nil {
		log.Fatalf("Req err: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+mhsToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Do err: %v", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func main() {
	config.LoadEnv()
	fmt.Println("=== UJI C.3: TEMPORARY FAILURE (MAILER_FAIL_EVERY=3) ===")

	// Step 1: Jalankan reset11.sql dan kosongkan logs/mail.log
	fmt.Println("0. Menjalankan reset11.sql dan mengosongkan logs/mail.log...")
	_ = os.WriteFile("logs/mail.log", []byte(""), 0644)

	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("DB connect: %v", err)
	}
	defer pool.Close()

	resetSQL, err := os.ReadFile("reset11.sql")
	if err != nil {
		log.Fatalf("Read reset11.sql: %v", err)
	}
	if _, err := pool.Exec(ctx, string(resetSQL)); err != nil {
		log.Fatalf("Exec reset11.sql: %v", err)
	}

	// 1. Daftar ke kelas 1, 2, 3, 4, 5, 6
	fmt.Println("1. Mendaftar ke kelas 1, 2, 3, 4, 5, 6 sebagai mhsuji...")
	for cid := 1; cid <= 6; cid++ {
		status := doReq(cid)
		fmt.Printf("   Kelas %d -> Status: %d\n", cid, status)
		if status != 201 {
			log.Fatalf("Gagal: kelas %d harus 201 tapi dapat %d", cid, status)
		}
	}

	fmt.Println("\n2. Menunggu 20 detik untuk retry Asynq menyelesaikan antrean...")
	for i := 1; i <= 4; i++ {
		time.Sleep(5 * time.Second)
		fmt.Printf("   Sudah %d detik...\n", i*5)
	}

	// 3. Verifikasi Database
	var dbNotifCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM notifications WHERE user_id = 7").Scan(&dbNotifCount)

	// 4. Verifikasi GET /notifications/mine
	req, _ := http.NewRequest("GET", baseURL+"/notifications/mine", nil)
	req.Header.Set("Authorization", "Bearer "+mhsToken)
	resp, _ := http.DefaultClient.Do(req)
	respBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var apiRes struct {
		Data []model.Notification `json:"data"`
	}
	json.Unmarshal(respBytes, &apiRes)
	apiNotifCount := len(apiRes.Data)

	// 5. Verifikasi logs/mail.log
	logContent, _ := os.ReadFile("logs/mail.log")
	mailLines := 0
	for _, l := range strings.Split(strings.TrimSpace(string(logContent)), "\n") {
		if strings.Contains(l, "mhsuji@") {
			mailLines++
		}
	}

	fmt.Printf("\n=== HASIL UJI C.3: TIGA ANGKA WAJIB SAMA-SAMA 6 ===\n")
	fmt.Printf("1. Jumlah baris notifications database: %d (Target: 6)\n", dbNotifCount)
	fmt.Printf("2. Jumlah notifikasi GET /notifications/mine: %d (Target: 6)\n", apiNotifCount)
	fmt.Printf("3. Jumlah baris logs/mail.log (mhsuji@): %d (Target: 6)\n", mailLines)

	if dbNotifCount == 6 && apiNotifCount == 6 && mailLines == 6 {
		fmt.Println(">>> UJI C.3 LOLOS SEMPURNA: 6 = 6 = 6! <<<")
	} else {
		fmt.Println(">>> UJI C.3 BELUM SESUAI SPESIFIKASI <<<")
	}
}

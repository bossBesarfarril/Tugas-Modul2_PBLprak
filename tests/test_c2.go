//go:build ignore

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"api-students/app/model"
	"api-students/config"
	"api-students/database"
)

const (
	mhsToken   = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6Im1oc3VqaSIsInJvbGUiOiJtYWhhc2lzd2EiLCJpc3MiOiJwcmFrdGlrdW0tYmFja2VuZCIsInN1YiI6IjciLCJleHAiOjE3OTMzNzg5MTgsImlhdCI6MTc5MzI5MjUxOH0.N7l2RzRLRggy3OCVuvoOWguumiYmggUjx2_H1Yt0IQw"
	adminToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoicHJha3Rpa3VtLWJhY2tlbmQiLCJzdWIiOiIzIiwiZXhwIjoxNzkzMzc4OTE4LCJpYXQiOjE3OTMyOTI1MTh9.hfOkBCfP0umHUTgbrdVAw7tpCqxaEC-06FVLhsf8oWQ"
	baseURL    = "http://localhost:3000"
)

func doReq(method, url, token, body string) (int, string, time.Duration) {
	start := time.Now()
	req, err := http.NewRequest(method, baseURL+url, strings.NewReader(body))
	if err != nil {
		log.Fatalf("Req err: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	duration := time.Since(start)
	if err != nil {
		log.Fatalf("Do err: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(respBody), duration
}

func main() {
	config.LoadEnv()
	fmt.Println("=== UJI C.2: KETAHANAN SAAT REDIS MATI ===")

	// 1. Matikan Redis
	fmt.Println("1. Menghentikan service/proses Redis...")
	_ = exec.Command("taskkill", "/F", "/IM", "redis-server.exe").Run()
	time.Sleep(2 * time.Second)

	// 2. Mendaftar kelas 4, 5, 6 saat Redis MATI
	fmt.Println("2. Mendaftar ke kelas 4, 5, dan 6 saat Redis mati (mengukur waktu tanggap)...")
	for _, cid := range []int{4, 5, 6} {
		status, body, dur := doReq("POST", fmt.Sprintf("/courses/%d/enroll", cid), mhsToken, "")
		fmt.Printf("   Kelas %d -> Status: %d | Waktu: %v\n   Body: %s\n", cid, status, dur, body)
		if status != 201 {
			log.Fatalf("Harus 201 tapi dapat: %d", status)
		}
	}

	// 3. Tunggu 5 detik, periksa GET /admin/queue
	fmt.Println("\n3. Menunggu 5 detik, lalu memeriksa GET /admin/queue...")
	time.Sleep(5 * time.Second)
	statusQ, bodyQ, _ := doReq("GET", "/admin/queue", adminToken, "")
	fmt.Printf("   GET /admin/queue -> Status: %d\n   Body: %s\n\n", statusQ, bodyQ)

	// 4. Nyalakan Redis kembali
	fmt.Println("4. Menyalakan kembali Redis server...")
	cmd := exec.Command("cmd", "/c", "start", `C:\laragon\bin\redis\redis-x64-5.0.14.1\redis-server.exe`)
	if err := cmd.Start(); err != nil {
		log.Fatalf("Gagal menyalakan Redis: %v", err)
	}
	fmt.Println("   Menunggu 10 detik agar relay mem-publish event yang tertahan...")
	time.Sleep(10 * time.Second)

	// 5. Periksa GET /admin/queue lagi
	fmt.Println("5. Memeriksa kembali GET /admin/queue setelah Redis aktif...")
	statusQ2, bodyQ2, _ := doReq("GET", "/admin/queue", adminToken, "")
	fmt.Printf("   GET /admin/queue -> Status: %d\n   Body: %s\n\n", statusQ2, bodyQ2)

	// 6. Periksa GET /notifications/mine milik mhsuji
	fmt.Println("6. Memeriksa GET /notifications/mine milik mhsuji (target: 7 notifikasi)...")
	statusM, bodyM, _ := doReq("GET", "/notifications/mine", mhsToken, "")
	var res struct {
		Success bool                 `json:"success"`
		Data    []model.Notification `json:"data"`
	}
	json.Unmarshal([]byte(bodyM), &res)
	fmt.Printf("   GET /notifications/mine -> Status: %d | Total Notifikasi: %d\n", statusM, len(res.Data))
	for i, n := range res.Data {
		fmt.Printf("   [%d] OutboxID: %d | Subject: %s\n", i+1, n.OutboxID, n.Subject)
	}

	// Verifikasi database outbox pending
	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("DB connect: %v", err)
	}
	defer pool.Close()

	var pendingCount int
	pool.QueryRow(ctx, "SELECT COUNT(*) FROM outbox WHERE published_at IS NULL").Scan(&pendingCount)

	fmt.Printf("\n=== HASIL UJI C.2 ===\n")
	fmt.Printf("Outbox Pending akhir: %d (Target: 0)\n", pendingCount)
	fmt.Printf("Total Notifikasi mhsuji: %d (Target: 7)\n", len(res.Data))
	if pendingCount == 0 && len(res.Data) == 7 {
		fmt.Println(">>> UJI C.2 LOLOS SEMPURNA! <<<")
	} else {
		fmt.Println(">>> UJI C.2 BELUM SESUAI SPESIFIKASI <<<")
	}
}

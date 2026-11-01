//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"api-students/app/model"
)

const (
	adminToken       = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIiwicm9sZSI6ImFkbWluIiwiaXNzIjoicHJha3Rpa3VtLWJhY2tlbmQiLCJzdWIiOiIzIiwiZXhwIjoxNzkzMzc4OTE4LCJpYXQiOjE3OTMyOTI1MTh9.hfOkBCfP0umHUTgbrdVAw7tpCqxaEC-06FVLhsf8oWQ"
	salahAlamatToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6InNhbGFoYWxhbWF0Iiwicm9sZSI6Im1haGFzaXN3YSIsImlzcyI6InByYWt0aWt1bS1iYWNrZW5kIiwic3ViIjoiOCIsImV4cCI6MTc5MzM3ODkxOCwiaWF0IjoxNzkzMjkyNTE4fQ.KRfqEMTbL-VtbLkm33r0A4eAl4hYQm_V6Se68QdCcFg"
	baseURL          = "http://localhost:3000"
)

func getQueueStats() model.QueueStats {
	req, _ := http.NewRequest("GET", baseURL+"/admin/queue", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Get queue: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	var res struct {
		Data model.QueueStats `json:"data"`
	}
	json.Unmarshal(b, &res)
	return res.Data
}

func main() {
	fmt.Println("=== UJI C.4: PERMANENT FAILURE (.invalid domain) ===")

	// 1. Catat archived awal
	statsAwal := getQueueStats()
	archivedAwal := 0
	if statsAwal.Queue != nil {
		archivedAwal = statsAwal.Queue.Archived
	}
	fmt.Printf("1. Keadaan queue awal -> Archived: %d | Retry: %d\n", archivedAwal, 0)

	// 2. Daftar ke kelas 2 sebagai salahalamat
	fmt.Println("2. Mendaftar ke kelas 2 sebagai salahalamat (salah@kampus.invalid)...")
	req, _ := http.NewRequest("POST", baseURL+"/courses/2/enroll", nil)
	req.Header.Set("Authorization", "Bearer "+salahAlamatToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Enroll: %v", err)
	}
	defer resp.Body.Close()
	fmt.Printf("   Status KRS: %d (Harap: 201)\n", resp.StatusCode)

	// 3. Tunggu 15 detik
	fmt.Println("3. Menunggu 15 detik untuk pemrosesan Asynq worker...")
	for i := 1; i <= 3; i++ {
		time.Sleep(5 * time.Second)
		fmt.Printf("   Sudah %d detik...\n", i*5)
	}

	// 4. Periksa queue akhir
	statsAkhir := getQueueStats()
	archivedAkhir := 0
	retryAkhir := 0
	if statsAkhir.Queue != nil {
		archivedAkhir = statsAkhir.Queue.Archived
		retryAkhir = statsAkhir.Queue.Retry
	}

	diffArchived := archivedAkhir - archivedAwal
	fmt.Printf("\n=== HASIL UJI C.4 ===\n")
	fmt.Printf("Pertambahan Queue Archived: %d (Target: TEPAT 1)\n", diffArchived)
	fmt.Printf("Queue Retry saat ini: %d (Target: 0)\n", retryAkhir)

	if diffArchived == 1 && retryAkhir == 0 {
		fmt.Println(">>> UJI C.4 LOLOS SEMPURNA: LANGSUNG ARCHIVED TANPA RETRY! <<<")
	} else {
		fmt.Println(">>> UJI C.4 BELUM SESUAI SPESIFIKASI <<<")
	}
}

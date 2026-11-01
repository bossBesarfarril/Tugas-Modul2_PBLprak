//go:build ignore

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"api-students/config"
	"api-students/database"
)

const (
	mhsToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6Im1oc3VqaSIsInJvbGUiOiJtYWhhc2lzd2EiLCJpc3MiOiJwcmFrdGlrdW0tYmFja2VuZCIsInN1YiI6IjciLCJleHAiOjE3OTMzNzg5MTgsImlhdCI6MTc5MzI5MjUxOH0.N7l2RzRLRggy3OCVuvoOWguumiYmggUjx2_H1Yt0IQw"
	baseURL  = "http://localhost:3000"
)

func doRequest(method, url, body string) (int, string) {
	req, err := http.NewRequest(method, baseURL+url, strings.NewReader(body))
	if err != nil {
		log.Fatalf("Create request failed: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+mhsToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(respBody)
}

func main() {
	config.LoadEnv()
	fmt.Println("=== UJI C.1: JALUR NORMAL ===")

	// 1. POST /courses/2/enroll
	status1, body1 := doRequest("POST", "/courses/2/enroll", "")
	fmt.Printf("[1] POST /courses/2/enroll -> Status: %d\nBody: %s\n\n", status1, body1)

	// 2. POST /courses/2/enroll (diulang) -> Harap 409
	status2, body2 := doRequest("POST", "/courses/2/enroll", "")
	fmt.Printf("[2] POST /courses/2/enroll (diulang) -> Status: %d\nBody: %s\n\n", status2, body2)

	// 3. DELETE /courses/2/enroll -> Harap 204
	status3, body3 := doRequest("DELETE", "/courses/2/enroll", "")
	fmt.Printf("[3] DELETE /courses/2/enroll -> Status: %d\nBody: %s\n\n", status3, body3)

	// 4. POST /courses/3/enroll -> Harap 201
	status4, body4 := doRequest("POST", "/courses/3/enroll", "")
	fmt.Printf("[4] POST /courses/3/enroll -> Status: %d\nBody: %s\n\n", status4, body4)

	// 5. POST /courses/3/swap with to_course_id: 1 -> Harap 200
	status5, body5 := doRequest("POST", "/courses/3/swap", `{"to_course_id": 1}`)
	fmt.Printf("[5] POST /courses/3/swap -> Status: %d\nBody: %s\n\n", status5, body5)

	fmt.Println("Menunggu 5 detik untuk worker & relay memproses seluruh antrean...")
	time.Sleep(5 * time.Second)

	// Verifikasi Database
	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("DB connect failed: %v", err)
	}
	defer pool.Close()

	fmt.Println("--- VERIFIKASI TABEL OUTBOX ---")
	rows, err := pool.Query(ctx, "SELECT id, event_type, published_at IS NOT NULL AS published FROM outbox ORDER BY id")
	if err != nil {
		log.Fatalf("Query outbox: %v", err)
	}
	defer rows.Close()
	outboxCount := 0
	for rows.Next() {
		var id int64
		var et string
		var pub bool
		rows.Scan(&id, &et, &pub)
		fmt.Printf("Outbox ID: %d | Event: %s | Published: %v\n", id, et, pub)
		outboxCount++
	}

	fmt.Println("\n--- VERIFIKASI TABEL NOTIFICATIONS ---")
	rowsN, err := pool.Query(ctx, "SELECT id, outbox_id, subject, sent_at FROM notifications ORDER BY id")
	if err != nil {
		log.Fatalf("Query notifications: %v", err)
	}
	defer rowsN.Close()
	notifCount := 0
	for rowsN.Next() {
		var id, outboxID int64
		var subj string
		var sent time.Time
		rowsN.Scan(&id, &outboxID, &subj, &sent)
		fmt.Printf("Notif ID: %d | Outbox ID: %d | Subject: %s | Sent: %s\n", id, outboxID, subj, sent.Format(time.RFC3339))
		notifCount++
	}

	fmt.Println("\n--- VERIFIKASI LOGS/MAIL.LOG ---")
	logContent, _ := os.ReadFile("logs/mail.log")
	logLines := strings.Split(strings.TrimSpace(string(logContent)), "\n")
	mailCount := 0
	for _, l := range logLines {
		if strings.Contains(l, "mhsuji@") {
			fmt.Println(l)
			mailCount++
		}
	}

	fmt.Printf("\n=== HASIL UJI C.1 ===\n")
	fmt.Printf("Outbox Events: %d (Target: 4)\n", outboxCount)
	fmt.Printf("Notifications: %d (Target: 4)\n", notifCount)
	fmt.Printf("Email Log Baris: %d (Target: 4)\n", mailCount)
	if outboxCount == 4 && notifCount == 4 && mailCount == 4 {
		fmt.Println(">>> UJI C.1 LOLOS SEMPURNA! <<<")
	} else {
		fmt.Println(">>> UJI C.1 BELUM LENGKAP <<<")
	}
}

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"

	"api-students/app/model"
	"api-students/config"
	"api-students/database"
)

func main() {
	config.LoadEnv()

	outboxID := flag.Int64("outbox", 0, "ID event outbox yang akan dikirim ulang")
	copies := flag.Int("copies", 5, "Jumlah salinan task yang dikirim ke queue")
	flag.Parse()

	if *outboxID <= 0 {
		log.Fatal("flag -outbox wajib diisi dengan ID yang valid")
	}

	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("gagal terhubung ke database: %v", err)
	}
	defer pool.Close()

	var eventType string
	var payload []byte
	err = pool.QueryRow(ctx, "SELECT event_type, payload FROM outbox WHERE id = $1", *outboxID).Scan(&eventType, &payload)
	if err != nil {
		log.Fatalf("mencari event outbox %d: %v", *outboxID, err)
	}

	job := model.NotificationJob{
		OutboxID:  *outboxID,
		EventType: eventType,
		Payload:   payload,
	}
	payloadBytes, err := json.Marshal(job)
	if err != nil {
		log.Fatalf("marshal job: %v", err)
	}

	client := asynq.NewClient(config.RedisOpt())
	defer client.Close()

	for i := 1; i <= *copies; i++ {
		// Task dibuat tanpa TaskID tetap agar kelima salinan berhasil masuk ke Redis
		task := asynq.NewTask(model.TaskSendNotification, payloadBytes,
			asynq.Queue(model.QueueNotifications),
			asynq.MaxRetry(5),
			asynq.Timeout(30*time.Second),
		)
		info, err := client.EnqueueContext(ctx, task)
		if err != nil {
			log.Fatalf("gagal enqueue salinan ke-%d: %v", i, err)
		}
		fmt.Printf("Salinan %d terkirim ke Redis (TaskID: %s)\n", i, info.ID)
	}

	fmt.Printf("Sukses: %d salinan task outbox %d berhasil dikirim ke antrean Redis\n", *copies, *outboxID)
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"api-students/app/model"
	"api-students/app/repository"
)

type OutboxRelay struct {
	outbox   repository.OutboxRepository
	tx       *repository.TxManager
	queue    *asynq.Client
	logger   *slog.Logger
	interval time.Duration
	batch    int
}

func NewOutboxRelay(
	outbox repository.OutboxRepository,
	tx *repository.TxManager,
	queue *asynq.Client,
	logger *slog.Logger,
	interval time.Duration,
	batch int,
) *OutboxRelay {
	return &OutboxRelay{
		outbox:   outbox,
		tx:       tx,
		queue:    queue,
		logger:   logger,
		interval: interval,
		batch:    batch,
	}
}

// Run menjalankan background loop relay.
func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		n, err := r.RelayOnce(ctx)
		if err != nil {
			r.logger.Warn("relay_delayed",
				slog.Int("published", n), slog.String("error", err.Error()))
		} else if n > 0 {
			r.logger.Info("relay_done", slog.Int("published", n))
		}

		if err == nil && n == r.batch && ctx.Err() == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RelayOnce mem-publish paling banyak satu batch. Mengembalikan jumlah
// event yang berhasil di-publish.
func (r *OutboxRelay) RelayOnce(ctx context.Context) (int, error) {
	var published int
	var enqueueErr error

	err := r.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		repo := r.outbox.WithTx(tx)

		events, err := repo.FetchUnpublished(ctx, r.batch)
		if err != nil {
			return err
		}

		ids := make([]int64, 0, len(events))
		for _, e := range events {
			// Berhenti pada kegagalan pertama (misal Redis mati).
			// Catatan D.1: Modul asli memanggil ids = append(ids, e.ID) SEBELUM r.publish.
			// Bila publish gagal, ID yang gagal tetap masuk ke ids dan ikut ditandai MarkPublished!
			// Perbaikan: Hanya append ID SETELAH r.publish sukses.
			if err := r.publish(ctx, e); err != nil {
				enqueueErr = err
				break
			}
			ids = append(ids, e.ID)
		}

		// Hanya event yang SUDAH berhasil di-enqueue yang ditandai.
		if err := repo.MarkPublished(ctx, ids); err != nil {
			return err
		}
		published = len(ids)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return published, enqueueErr
}

func (r *OutboxRelay) publish(ctx context.Context, e model.OutboxEvent) error {
	job := model.NotificationJob{
		OutboxID:  e.ID,
		EventType: e.EventType,
		Payload:   e.Payload,
	}

	// Catatan D.1: Modul asli mengirimkan job (struct) langsung ke asynq.NewTask,
	// yang menyebabkan compiler error karena NewTask membutuhkan []byte.
	// Perbaikan: Lakukan json.Marshal terlebih dahulu.
	payloadBytes, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}

	task := asynq.NewTask(model.TaskSendNotification, payloadBytes,
		// TaskID yang diturunkan dari id outbox membuat enqueue ulang
		// event yang sama ditolak Redis. Ini menutup celah ketika relay
		// mati SETELAH melakukan enqueue tetapi SEBELUM sempat commit.
		asynq.TaskID(fmt.Sprintf("outbox-%d", e.ID)),
		asynq.Queue(model.QueueNotifications),
		asynq.MaxRetry(5),
		asynq.Timeout(30*time.Second),
		// Task yang selesai disimpan 24 jam.
		asynq.Retention(24*time.Hour),
	)

	_, err = r.queue.EnqueueContext(ctx, task)
	if errors.Is(err, asynq.ErrTaskIDConflict) {
		// Sudah pernah di-enqueue pada putaran sebelumnya. Bagi relay, itu sama dengan berhasil.
		return nil
	}
	if err != nil {
		return fmt.Errorf("enqueue outbox %d: %w", e.ID, err)
	}
	return nil
}

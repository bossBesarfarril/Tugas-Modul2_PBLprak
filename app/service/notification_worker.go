package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"

	"api-students/app/model"
	"api-students/app/repository"
	"api-students/helper"
)

type NotificationWorker struct {
	users         repository.UserRepository
	courses       repository.CourseRepository
	notifications repository.NotificationRepository
	tx            *repository.TxManager
	mailer        helper.Mailer
	logger        *slog.Logger
}

func NewNotificationWorker(
	users repository.UserRepository,
	courses repository.CourseRepository,
	notifications repository.NotificationRepository,
	tx *repository.TxManager,
	mailer helper.Mailer,
	logger *slog.Logger,
) *NotificationWorker {
	return &NotificationWorker{
		users:         users,
		courses:       courses,
		notifications: notifications,
		tx:            tx,
		mailer:        mailer,
		logger:        logger,
	}
}

// HandleSend mengerjakan task "notification:send".
// Catatan D.1: Modul asli menerima t asynq.Task (value), yang menyebabkan error compiler
// saat didaftarkan ke asynq.ServeMux. Perbaikan: gunakan pointer t *asynq.Task.
func (w *NotificationWorker) HandleSend(ctx context.Context, t *asynq.Task) error {
	var job model.NotificationJob
	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("%w: payload task rusak: %v", asynq.SkipRetry, err)
	}

	// Penanganan Tugas Mandiri D.2: Notifikasi Perubahan Kuota untuk Seluruh Peserta
	if job.EventType == model.EventCourseQuotaChanged {
		return w.handleQuotaChanged(ctx, job)
	}

	// Penanganan normal untuk pendaftaran KRS (Enroll, Drop, Swap)
	var event model.EnrollmentEvent
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload event %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}

	user, err := w.users.FindByID(ctx, event.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("%w: user %d tidak ditemukan", asynq.SkipRetry, event.UserID)
	}
	if err != nil {
		return err // database bermasalah: layak di-retry
	}

	msg, err := BuildMessage(job.EventType, event, user)
	if err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	alreadySent := false
	err = w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		// Catatan D.1: Modul asli memanggil w.notifications.Claim tanpa WithTx(tx).
		// Perbaikan: gunakan w.notifications.WithTx(tx).Claim agar berada di dalam transaksi yang sama.
		claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
			OutboxID: job.OutboxID,
			UserID:   user.ID,
			Channel:  "email",
			Subject:  msg.Subject,
			Body:     msg.Body,
		})
		if err != nil {
			return err
		}
		if !claimed {
			alreadySent = true
			return nil
		}
		return w.mailer.Send(ctx, msg)
	})

	// Catatan D.1: Modul asli menulis fmt.Errorf("%v: %w", asynq.SkipRetry, err)
	// sehingga asynq.SkipRetry TIDAK terbungkus oleh %w dan Asynq tetap melakukan retry!
	// Perbaikan: gunakan fmt.Errorf("%w: %v", asynq.SkipRetry, err).
	if errors.Is(err, helper.ErrPermanentDelivery) {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	if err != nil {
		return err
	}

	retry, _ := asynq.GetRetryCount(ctx)
	w.logger.Info("notification_processed",
		slog.Int64("outbox_id", job.OutboxID),
		slog.String("event", job.EventType),
		slog.Int("retry", retry),
		slog.Bool("already_sent", alreadySent))
	return nil
}

// handleQuotaChanged mengirim email perubahan kuota ke seluruh peserta kelas (Tugas Mandiri D.2).
// Idempoten per-mahasiswa: bila pengiriman ke mahasiswa ke-N gagal, retry tidak akan
// mengirim ulang ke mahasiswa yang sudah sukses dikirimi.
func (w *NotificationWorker) handleQuotaChanged(ctx context.Context, job model.NotificationJob) error {
	var qc model.QuotaChangedEvent
	if err := json.Unmarshal(job.Payload, &qc); err != nil {
		return fmt.Errorf("%w: payload quota rusak: %v", asynq.SkipRetry, err)
	}

	userIDs, err := w.courses.FindEnrolledUsers(ctx, qc.CourseID)
	if err != nil {
		return fmt.Errorf("mengambil peserta kelas %d: %w", qc.CourseID, err)
	}

	retry, _ := asynq.GetRetryCount(ctx)
	for _, uid := range userIDs {
		user, err := w.users.FindByID(ctx, uid)
		if err != nil {
			continue
		}

		msg := BuildQuotaMessage(qc, user)
		alreadySent := false

		err = w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
			claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
				OutboxID: job.OutboxID,
				UserID:   user.ID,
				Channel:  "email",
				Subject:  msg.Subject,
				Body:     msg.Body,
			})
			if err != nil {
				return err
			}
			if !claimed {
				alreadySent = true
				return nil
			}
			return w.mailer.Send(ctx, msg)
		})

		if errors.Is(err, helper.ErrPermanentDelivery) {
			// Lewati mahasiswa dengan email permanent failure agar mahasiswa lain tetap terkirim
			w.logger.Warn("quota_mail_permanent_fail", slog.String("email", user.Email))
			continue
		}
		if err != nil {
			// Temporary failure: gagalkan task agar di-retry Asynq
			return err
		}

		w.logger.Info("notification_processed",
			slog.Int64("outbox_id", job.OutboxID),
			slog.String("event", job.EventType),
			slog.Int("user_id", uid),
			slog.Int("retry", retry),
			slog.Bool("already_sent", alreadySent))
	}

	return nil
}

-- Migration 009: Idempotency Key Komposit untuk Notifikasi Siaran (Broadcast)
--   psql -h 127.0.0.1 -p 5432 -U koleksi_admin -d api_students -f migrations/009_quota_notification.sql

-- Lepas constraint UNIQUE lama yang hanya berbasis outbox_id tunggal
ALTER TABLE notifications 
    DROP CONSTRAINT IF EXISTS notifications_outbox_unique;

-- Buat constraint UNIQUE komposit (outbox_id, user_id)
-- Memungkinkan 1 outbox broadcast menghasilkan notifikasi ke banyak mahasiswa,
-- sekaligus menjamin idempotensi per-mahasiswa saat terjadi retry.
ALTER TABLE notifications 
    DROP CONSTRAINT IF EXISTS notifications_outbox_user_unique;

ALTER TABLE notifications 
    ADD CONSTRAINT notifications_outbox_user_unique UNIQUE (outbox_id, user_id);

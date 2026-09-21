-- Untuk mendukung pencarian cursor dengan cepat, kita buat index gabungan
-- dengan urutan sesuai DESCENDING.
CREATE INDEX idx_students_created_id ON students (created_at DESC, id DESC);
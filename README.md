# API Students (Modul 3 - PostgreSQL & Repository Pattern)

REST API untuk mengelola data mahasiswa dengan database PostgreSQL menggunakan pola Repository Pattern.

## Persyaratan Sistem

- Go 1.22 atau lebih baru
- PostgreSQL 15 atau lebih baru
- Git

## Daftar Variabel Environment (`.env`)

Aplikasi ini membutuhkan konfigurasi environment berikut yang disimpan di berkas `.env` pada folder root:

```env
APP_PORT=3000

DB_HOST=127.0.0.1
DB_PORT=5432
DB_USER=koleksi_admin
DB_PASSWORD=
DB_NAME=api_students
DB_SSLMODE=disable
DB_MAX_CONNS=10
```

*Catatan: Pastikan menggunakan `DB_HOST=127.0.0.1` jika user `koleksi_admin` menggunakan metode autentikasi `trust`.*

---

## Cara Menyiapkan Database dari Nol

### 1. Buat Database
Buka terminal dan buat database bernama `api_students` menggunakan user `koleksi_admin` dengan perintah berikut:
```bash
psql -h 127.0.0.1 -U koleksi_admin -d postgres -c "CREATE DATABASE api_students"
```

### 2. Jalankan Migrasi Tabel
Jalankan berkas migrasi SQL untuk membuat tabel dan indeks:
```bash
psql -h 127.0.0.1 -U koleksi_admin -d api_students -f migrations/001_create_students.sql
```

---

## Skema Tabel `students`

Tabel `students` memiliki struktur kolom dan indeks berikut:

```sql
CREATE TABLE IF NOT EXISTS students (
    id         SERIAL        PRIMARY KEY,
    nim        VARCHAR(20)   NOT NULL,
    name       VARCHAR(100)  NOT NULL,
    grade      NUMERIC(5,2)  NOT NULL DEFAULT 0,
    is_active  BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
```

### Indeks (Indexes):
1.  `PRIMARY KEY (id)`: Indeks otomatis pada kolom ID.
2.  `students_nim_lower_key` (`UNIQUE INDEX`): Menjamin keunikan NIM (case-insensitive) menggunakan `LOWER(nim)`.
3.  `students_name_lower_idx` (`INDEX`): Mengoptimalkan kecepatan query pencarian nama mahasiswa menggunakan `LOWER(name)`.

---

## Kontrak API (Endpoints)

### Base URL
```
/api/v1
```

### Daftar Endpoints

| Metode | Endpoint | Deskripsi |
|--------|----------|-----------|
| GET | `/api/v1/students` | Daftar semua student (dengan paginasi) |
| GET | `/api/v1/students/:id` | Ambil satu student berdasarkan ID |
| POST | `/api/v1/students` | Tambah student baru |
| PUT | `/api/v1/students/:id` | Ganti seluruh data student |
| PATCH | `/api/v1/students/:id` | Ubah sebagian data student |
| DELETE | `/api/v1/students/:id` | Hapus student |

---

## Cara Menjalankan Aplikasi

1. Unduh semua dependensi proyek:
   ```bash
   go mod tidy
   ```
2. Jalankan server:
   ```bash
   go run .
   ```
3. Server akan berjalan di `http://localhost:3000`
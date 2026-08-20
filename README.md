# API Students

REST API untuk mengelola data mahasiswa, dibuat dengan Go Fiber.

## Cara Menjalankan

```bash
go run .
```

Server berjalan di `http://localhost:3000`

## Kontrak API

### Base URL

```
/api/v1
```

### Endpoints

| Metode | Endpoint | Deskripsi |
|--------|----------|-----------|
| GET | `/api/v1/students` | Daftar semua student (dengan paginasi) |
| GET | `/api/v1/students/:id` | Ambil satu student berdasarkan ID |
| POST | `/api/v1/students` | Tambah student baru |
| PUT | `/api/v1/students/:id` | Ganti seluruh data student |
| PATCH | `/api/v1/students/:id` | Ubah sebagian data student |
| DELETE | `/api/v1/students/:id` | Hapus student |

---

### GET `/api/v1/students`

**Query Parameters:**

| Parameter | Tipe | Default | Keterangan |
|-----------|------|---------|------------|
| page | int | 1 | Halaman keberapa |
| limit | int | 10 | Jumlah per halaman (maks 50) |
| search | string | - | Cari berdasarkan nama (case-insensitive) |
| sort | string | id | Urutkan berdasarkan: `id`, `nim`, `name`, `grade` |
| order | string | asc | Arah urutan: `asc` atau `desc` |
| is_active | bool | - | Filter berdasarkan status aktif |

**Contoh Request:**
```
GET /api/v1/students?page=1&limit=10&search=budi&sort=name&order=asc&is_active=true
```

**Contoh Response (200):**
```json
{
  "success": true,
  "message": "daftar student berhasil diambil",
  "data": [
    {
      "id": 1,
      "nim": "2024001",
      "name": "Budi Santoso",
      "grade": 85.5,
      "is_active": true
    }
  ],
  "meta": {
    "page": 1,
    "limit": 10,
    "total": 1,
    "total_pages": 1
  }
}
```

---

### GET `/api/v1/students/:id`

**Contoh Response (200):**
```json
{
  "success": true,
  "message": "student ditemukan",
  "data": {
    "id": 1,
    "nim": "2024001",
    "name": "Budi Santoso",
    "grade": 85.5,
    "is_active": true
  }
}
```

**Status:** 200, 400 (id bukan angka), 404 (tidak ditemukan)

---

### POST `/api/v1/students`

**Contoh Body:**
```json
{
  "nim": "2024001",
  "name": "Budi Santoso",
  "grade": 85.5
}
```

**Contoh Response (201):**
```json
{
  "success": true,
  "message": "student berhasil dibuat",
  "data": {
    "id": 1,
    "nim": "2024001",
    "name": "Budi Santoso",
    "grade": 85.5,
    "is_active": true
  }
}
```

**Header Response:** `Location: /api/v1/students/1`

**Status:** 201, 400 (JSON tidak valid), 409 (NIM duplikat), 415 (bukan JSON), 422 (validasi gagal)

---

### PUT `/api/v1/students/:id`

Mengganti **seluruh** data student. Semua field wajib dikirim.

**Contoh Body:**
```json
{
  "nim": "2024001",
  "name": "Budi Santoso Edited",
  "grade": 90.0,
  "is_active": false
}
```

**Status:** 200, 400, 404, 409, 415, 422

---

### PATCH `/api/v1/students/:id`

Mengubah **sebagian** data student. Hanya kirim field yang ingin diubah.

**Contoh Body:**
```json
{
  "grade": 95.0
}
```

**Status:** 200, 400, 404, 409, 415, 422

---

### DELETE `/api/v1/students/:id`

Menghapus student. Tidak mengembalikan body (204 No Content).

**Status:** 204, 400, 404

---

## Daftar Status HTTP

| Status | Nama | Situasi |
|--------|------|---------|
| 200 | OK | Pengambilan atau perubahan berhasil |
| 201 | Created | Penambahan berhasil, disertai header Location |
| 204 | No Content | Penghapusan berhasil |
| 400 | Bad Request | Body bukan JSON valid, atau id bukan angka |
| 404 | Not Found | Data tidak ditemukan |
| 409 | Conflict | NIM sudah dipakai (duplikat) |
| 415 | Unsupported Media Type | Content-Type bukan application/json |
| 422 | Unprocessable Entity | Validasi isi gagal, dengan rincian per field |

## Contoh Response Gagal

**404 - Tidak Ditemukan:**
```json
{
  "success": false,
  "message": "student tidak ditemukan"
}
```

**422 - Validasi Gagal:**
```json
{
  "success": false,
  "message": "validasi gagal",
  "errors": {
    "nim": "wajib diisi",
    "name": "wajib diisi"
  }
}
```

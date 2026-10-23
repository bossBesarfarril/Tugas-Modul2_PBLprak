# LAPORAN PRAKTIKUM PEMROGRAMAN BACKEND LANJUT
**Modul 10 — API dengan MongoDB: Implementasi Document Model, Atomic Update, dan Polyglot Persistence**

---

**Disusun Oleh:**

Nama: Muhammad Farril Nasywa  
NIM: 434241124  
Kelas: C4 / D4 Teknik Informatika  
Repository GitHub: `[Masukkan Link Repository GitHub Anda di Sini]` *(Pastikan .env tidak terunggah)*

**FAKULTAS VOKASI**  
**UNIVERSITAS AIRLANGGA**  
**2026**

---

## Daftar Isi
- [Kata Pengantar](#kata-pengantar)
- [BAB I: Pendahuluan](#bab-i)
  - 1.1 Latar Belakang
  - 1.2 Tujuan Pembelajaran
  - 1.3 Alat dan Lingkungan Praktikum
- [BAB II: Pembahasan dan Langkah Pengerjaan](#bab-ii)
  - 2.1 Arsitektur Polyglot Persistence (PostgreSQL & MongoDB)
  - 2.2 Rancangan Dokumen & Indeks Koleksi (student_portfolios)
  - 2.3 Implementasi Lapisan Kode (Clean Architecture)
  - 2.4 Hasil Pengujian Endpoint dan Kasus Uji
- [BAB III: Analisis dan Evaluasi](#bab-iii)
  - 3.1 Analisis Desain Embedding vs Referencing
  - 3.2 Penanganan Konsistensi Data Lintas Database (PostgreSQL & MongoDB)
  - 3.3 Analisis Ketersediaan Sistem saat Database Mengalami Kegagalan
- [BAB IV: Penutup](#bab-iv)
  - 4.1 Kesimpulan
  - 4.2 Saran

---

## Kata Pengantar

Puji syukur kehadirat Tuhan Yang Maha Esa atas berkat dan rahmat-Nya, penyusunan laporan praktikum Pemrograman Backend Lanjut "Modul 10 — API dengan MongoDB" ini dapat diselesaikan dengan baik dan tepat waktu.

Laporan ini menyajikan dokumentasi teknis penerapan konsep *Polyglot Persistence*, di mana dua sistem basis data berbeda (PostgreSQL sebagai sistem relasional dan MongoDB sebagai sistem dokumen BSON) diintegrasikan dalam satu arsitektur aplikasi Go Fiber. Praktikum ini memberikan wawasan mendalam mengenai penggunaan pembaruan atomik (`$set`, `$push`, `$pull`), pengelolaan indeks unik untuk mencegah duplikasi, pemanfaatan *Aggregation Pipeline*, serta pertahanan terhadap serangan *NoSQL Injection*.

Ucapan terima kasih kami sampaikan kepada dosen pengampu mata kuliah Praktikum Pemrograman Backend Lanjut serta asisten praktikum atas bimbingan dan arahan yang telah diberikan. Penulis menyadari masih terdapat ruang perbaikan, sehingga saran yang membangun sangat diharapkan demi kesempurnaan di masa mendatang.

Surabaya, Oktober 2026

Muhammad Farril Nasywa  
(NIM: 434241124)

---

## BAB I
## Pendahuluan

### 1.1 Latar Belakang
Pada pengembangan perangkat lunak berskala industri modern, tidak ada satu jenis database yang ideal untuk segala jenis data (*no one-size-fits-all*). Database relasional seperti PostgreSQL sangat unggul dalam menjaga konsistensi transaksi (ACID), integritas referensial (foreign keys), dan batasan kaku. Namun, untuk data yang bersifat dinamis, memiliki hierarki bersarang (*nested document*), serta memiliki struktur atribut yang fleksibel seperti data portofolio mahasiswa (proyek, keahlian, dan sertifikat), model dokumen NoSQL seperti MongoDB menawarkan fleksibilitas serta performa pembacaan yang jauh lebih efisien tanpa memerlukan operasi *JOIN* tabel yang mahal.

Penerapan *Polyglot Persistence*—menggabungkan PostgreSQL dan MongoDB dalam satu layanan backend—menjadi jawaban atas kebutuhan tersebut. Modul praktikum ini mengimplementasikan pengelolaan portofolio mahasiswa pada entitas `students` yang terintegrasi antara data induk di PostgreSQL dan data portofolio kaya-fitur di MongoDB.

### 1.2 Tujuan Pembelajaran
1. Mahasiswa mampu menghubungkan aplikasi Go/Fiber ke MongoDB menggunakan driver resmi `go.mongodb.org/mongo-driver/v2`.
2. Mahasiswa mampu merancang *document model* pada MongoDB dengan relasi *embedding* dan *referencing*.
3. Mahasiswa mampu menerapkan operasi pembaruan atomik (`$set`, `$push`, `$pull`) yang kebal terhadap *race condition*.
4. Mahasiswa mampu merancang indeks unik pada koleksi MongoDB dan memverifikasi kinerjanya menggunakan `explain("executionStats")`.
5. Mahasiswa mampu menyusun *Aggregation Pipeline* dengan operator `$unwind` untuk menyajikan data analitik statistik.
6. Mahasiswa memahami dan mampu mendemonstrasikan pertahanan terhadap kerentanan *NoSQL Injection*.

### 1.3 Alat dan Lingkungan Praktikum
1. **Bahasa Pemrograman:** Go (Golang) v1.26
2. **Framework Web:** Go Fiber v2
3. **Database RDBMS:** PostgreSQL 18 (Relational data `students`)
4. **Database NoSQL:** MongoDB Community Server v8.3.7 & MongoDB Compass
5. **Driver Go:** `go.mongodb.org/mongo-driver/v2`
6. **Alat Pengujian API:** Postman Desktop & Terminal CLI

---

## BAB II
## Pembahasan dan Langkah Pengerjaan

### 2.1 Arsitektur Polyglot Persistence (PostgreSQL & MongoDB)
Sistem mempertahankan pemisahan tanggung jawab yang jelas:
* Data entitas inti mahasiswa (`id`, `nim`, `name`, `grade`, `owner_id`) dikelola secara ketat di **PostgreSQL**.
* Data portofolio mahasiswa (`headline`, `bio`, array `projects`, array `certificates`) disimpan di **MongoDB** pada koleksi `student_portfolios`.
* Layer **Service** (`PortfolioService`) bertindak sebagai koordinator orkestrasi: sebelum melakukan modifikasi atau pembacaan data portofolio ke MongoDB, sistem melakukan verifikasi ke PostgreSQL untuk memastikan mahasiswa dengan ID yang bersangkutan benar-benar terdaftar di basis data kampus.

### 2.2 Rancangan Dokumen & Indeks Koleksi (student_portfolios)
Setiap mahasiswa memiliki tepat satu dokumen portofolio. Kepemilikan tunggal ini ditegakkan langsung di level database MongoDB menggunakan **Unique Index** pada atribut `student_id` (nama indeks: `uniq_student_id`).

#### Contoh Satu Dokumen Lengkap Portofolio (Keluaran BSON/JSON):
```json
{
  "_id": "6adafa19e83c2b1df79c7a32",
  "student_id": 7,
  "headline": "Junior Go Backend Developer & Cloud Enthusiast",
  "bio": "Mahasiswa D4 Teknik Informatika UNAIR yang berfokus pada arsitektur microservices dan database NoSQL.",
  "projects": [
    {
      "id": "6adafa1939bd2ad0fdh32654",
      "title": "SIAKAD REST API with Go Fiber",
      "technologies": ["Go", "Fiber", "PostgreSQL", "MongoDB"],
      "link": "https://github.com/farril/siakad-api",
      "year": 2026
    },
    {
      "id": "6adafa76e83c2b1df79c7a34",
      "title": "NFC Attendance System",
      "technologies": ["Go", "MQTT", "PostgreSQL", "Docker"],
      "link": "https://github.com/farril/nfc-absensi",
      "year": 2026
    }
  ],
  "certificates": [
    {
      "name": "MongoDB Certified Developer",
      "issuer": "MongoDB University",
      "date": "2026-10-15"
    }
  ],
  "created_at": "2026-10-23T06:09:29.288Z",
  "updated_at": "2026-10-23T06:24:25.768Z"
}
```

> **[TEMPATKAN SCREENSHOT COMPASS DI SINI]**  
> *Gambar 1: Tangkapan layar MongoDB Compass menampilkan dokumen portofolio mahasiswa lengkap dengan struktur array projects dan certificates yang telah terisi.*

---

### 2.3 Implementasi Lapisan Kode (Clean Architecture)
Sesuai prinsip *Clean Architecture*, layer `model` murni tidak boleh terikat (*decoupled*) dengan dependensi BSON pihak ketiga. Pemetaan tag BSON dilakukan secara internal di dalam struct privat pada layer `repository`.

#### A. Inisialisasi Koneksi MongoDB (`database/mongo.go`):
```go
func NewMongoClient(ctx context.Context) (*mongo.Client, error) {
    opts := options.Client().
        ApplyURI(config.GetEnv("MONGO_URI", "mongodb://localhost:27017")).
        SetServerSelectionTimeout(5 * time.Second).
        SetMaxPoolSize(uint64(config.GetEnvInt("MONGO_MAX_POOL", 20)))

    client, err := mongo.Connect(opts)
    if err != nil {
        return nil, fmt.Errorf("menyiapkan client MongoDB: %w", err)
    }

    pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    if err := client.Ping(pingCtx, readpref.Primary()); err != nil {
        _ = client.Disconnect(context.Background())
        return nil, fmt.Errorf("ping MongoDB: %w", err)
    }
    return client, nil
}
```

#### B. Logika Atomik Pembatasan Maksimal 20 Proyek (`app/repository/portofolio_repository.go`):
Untuk menghindari *race condition* saat banyak permintaan penambahan proyek masuk bersamaan, sistem menerapkan filter bersyarat langsung pada kueri pembaruan MongoDB menggunakan selektor `"projects.19": {"$exists": false}`:
```go
filter := bson.M{
    "student_id":  studentID,
    "projects.19": bson.M{"$exists": false}, // Memastikan indeks ke-19 belum ada (maks 20 proyek)
}
update := bson.M{
    "$push": bson.M{"projects": newProj},
    "$set":  bson.M{"updated_at": now()},
}
```

#### C. Aggregation Pipeline 10 Teknologi Populer (`app/repository/portofolio_repository.go`):
Pencarian 10 teknologi paling dominan membutuhkan dua kali tahap pemecahan array (`$unwind`):
```go
pipeline := mongo.Pipeline{
    bson.D{{Key: "$unwind", Value: "$projects"}},
    bson.D{{Key: "$unwind", Value: "$projects.technologies"}},
    bson.D{{Key: "$group", Value: bson.D{
        {Key: "_id", Value: "$projects.technologies"},
        {Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
    }}},
    bson.D{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
    bson.D{{Key: "$limit", Value: 10}},
    bson.D{{Key: "$project", Value: bson.D{
        {Key: "_id", Value: 0},
        {Key: "technology", Value: "$_id"},
        {Key: "count", Value: 1},
    }}},
}
```

---

### 2.4 Hasil Pengujian Endpoint dan Kasus Uji

#### 1. Uji Health Check Endpoint (`GET /api/v1/health`)
Endpoint memastikan kedua server database (PostgreSQL dan MongoDB) terhubung dan sehat secara bersamaan.
> **[TEMPATKAN SCREENSHOT HEALTH CHECK DI SINI]**  
> *Gambar 2: Pengujian Endpoint Health Check menghasilkan status 200 OK dengan pesan "server, PostgreSQL, dan MongoDB berjalan".*

#### 2. Uji Upsert Portofolio (`PUT /api/v1/students/:id/portfolio`)
Digunakan untuk menginisialisasi atau memperbarui keseluruhan portofolio mahasiswa.
> **[TEMPATKAN SCREENSHOT PUT PORTOFOLIO DI SINI]**  
> *Gambar 3: Pengujian PUT /students/7/portfolio menghasilkan respon 200 OK dan menyimpan dokumen portofolio mahasiswa ID 7.*

#### 3. Uji Membaca Portofolio (`GET /api/v1/students/:id/portfolio`)
Digunakan untuk mengambil seluruh data portofolio mahasiswa beserta proyek dan sertifikat yang bersarang di dalamnya.
> **[TEMPATKAN SCREENSHOT GET PORTOFOLIO DI SINI]**  
> *Gambar 4: Pengujian GET /students/7/portfolio menampilkan seluruh data portofolio secara komprehensif.*

#### 4. Uji Penambahan Proyek Atomik (`POST /api/v1/students/:id/portfolio/projects`)
Membuktikan penambahan satu elemen baru ke dalam array `projects` menggunakan operator `$push`.
> **[TEMPATKAN SCREENSHOT POST PROYEK DI SINI]**  
> *Gambar 5: Pengujian POST /students/7/portfolio/projects berhasil menambahkan proyek kedua dengan status 201 Created.*

#### 5. Uji Penghapusan Proyek Atomik (`DELETE /api/v1/students/:id/portfolio/projects/:projectId`)
Membuktikan penghapusan elemen proyek spesifik berdasarkan ObjectID menggunakan operator `$pull`.
> **[TEMPATKAN SCREENSHOT DELETE PROYEK DI SINI]**  
> *Gambar 6: Pengujian DELETE proyek menghasilkan status 200 OK dengan pesan "proyek berhasil dihapus".*

#### 6. Uji Statistik Agregasi Teknologi (`GET /api/v1/portfolios/stats/technologies`)
Membuktikan keluaran *Aggregation Pipeline* 10 teknologi terpopuler yang dipakai mahasiswa.
> **[TEMPATKAN SCREENSHOT STATISTIK AGREGASI DI SINI]**  
> *Gambar 7: Pengujian GET /portfolios/stats/technologies mengembalikan daftar peringkat frekuensi teknologi yang digunakan.*

---

### 2.5 Pembuktian Pengujian Khusus (Kasus Gagal, Indexing, dan Concurrency)

#### A. Pembuktian Tiga Kasus Gagal yang Berbeda:
1. **Kasus Gagal 1: Mahasiswa Tidak Terdaftar di PostgreSQL (Status 404 Not Found)**  
   Ketika mencoba membuat portofolio untuk `id: 1` yang tidak ada di database relasional PostgreSQL, sistem menolak transaksi dengan pesan `"mahasiswa tidak ditemukan di sistem"`.
   > **[TEMPATKAN SCREENSHOT KASUS GAGAL 1 (404 NOT FOUND) DI SINI]**  
   > *Gambar 8: Penolakan akses saat ID mahasiswa tidak valid di PostgreSQL.*

2. **Kasus Gagal 2: Penolakan Serangan NoSQL Injection (Status 400 Bad Request)**  
   Pengujian keamanan dilakukan dengan menyuntikkan operator kueri MongoDB `{"$ne": ""}` pada field JSON `title`. Sistem menolak kueri tersebut secara otomatis pada tahap parsing struct Go dengan pesan `"format body json tidak valid"`.
   > **[TEMPATKAN SCREENSHOT KASUS GAGAL 2 (NOSQL INJECTION) DI SINI]**  
   > *Gambar 9: Bukti pertahanan sistem terhadap NoSQL Injection.*

3. **Kasus Gagal 3: Penolakan Format ID Proyek Tidak Sah pada DELETE (Status 400 Bad Request)**  
   Ketika endpoint DELETE dipanggil dengan string ID acak yang bukan merupakan representasi valid dari 12-byte Hexadecimal BSON ObjectID, repository langsung menolaknya sebelum kueri dieksekusi.

#### B. Pembuktian Eksekusi Index Menggunakan Explain Plan (`EXPRESS_IXSCAN`):
Verifikasi indeks dilakukan melalui MongoDB Compass dengan mengevaluasi filter `{ student_id: 7 }`. Hasil pengujian membuktikan bahwa MongoDB mengeksekusi tahapan **`EXPRESS_IXSCAN`** menggunakan indeks **`uniq_student_id`** dengan durasi eksekusi **`0 ms`**, membuktikan kueri terindeks secara optimal tanpa melakukan pemindaian seluruh koleksi (`COLLSCAN`).
> **[TEMPATKAN SCREENSHOT EXPLAIN PLAN COMPASS DI SINI]**  
> *Gambar 10: Hasil Explain Plan membuktikan eksekusi IXSCAN menggunakan indeks uniq_student_id.*

#### C. Pembuktian Uji Serangan Serentak (Batas 20 Proyek Bebas Race Condition):
Pengujian konkurensi dilakukan dengan menyiapkan 18 proyek awal, kemudian menembakkan 10 request `POST` penambahan proyek secara serentak di milidetik yang sama menggunakan *Goroutines*. Hasil pengujian membuktikan:
* Tepat **2 request berhasil** (Status 201 Created), menggenapkan kapasitas proyek menjadi pas 20.
* Tepat **8 request lainnya langsung ditolak** (Status 400 Bad Request).
* Total proyek di basis data akhir tetap terjaga pada angka **20 proyek**, membuktikan keandalan logika atomik dari serangan *Race Condition*.
> **[TEMPATKAN SCREENSHOT TERMINAL UJI CONCURRENCY DI SINI]**  
> *Gambar 11: Bukti keluaran skrip uji konkurensi membuktikan batas 20 proyek terjaga secara atomik.*

---

## BAB III
## ANALISIS DAN EVALUASI

*(Jawaban resmi Tugas Mandiri C.4 merujuk pada rancangan sistem)*

### 3.1 Analisis Desain Embedding vs Referencing
Array `projects` disarangkan (*embedded*) langsung di dalam dokumen `student_portfolios` karena data proyek memiliki rasio pembacaan yang selalu terikat erat dengan portofolio induknya (*tightly-coupled*), memiliki batas kardinalitas yang terprediksi (dibatasi maksimal 20 item), dan bertujuan untuk meminimalkan beban komputasi server dengan menghilangkan kebutuhan operasi `$lookup` atau kueri bertingkat saat portofolio ditampilkan. Namun, keputusan ini harus dibalik (diubah menjadi *referencing* ke koleksi tersendiri) apabila muncul kebutuhan baru di mana sebuah proyek dapat dikerjakan secara kolaboratif oleh banyak mahasiswa (relasi *many-to-many*) atau ketika ukuran deskripsi proyek serta lampiran aset multimedia di dalamnya sangat besar sehingga berpotensi menabrak batas kapasitas fisik dokumen MongoDB sebesar 16 Megabyte.

### 3.2 Penanganan Konsistensi Data Lintas Database (PostgreSQL & MongoDB)
Bila data seorang mahasiswa dihapus dari PostgreSQL, portofolionya di MongoDB akan menjadi data yatim (*orphaned document*) karena ketiadaan mekanisme *Foreign Key CASCADE* otomatis lintas engine database yang berbeda. Cara penanganan pertama adalah menerapkan **Pembersihan Asinkron Berbasis Event (*Event-Driven Cleanup*)**: saat penghapusan mahasiswa terjadi di PostgreSQL, aplikasi menerbitkan pesan ke *message broker* (misalnya RabbitMQ atau Kafka) yang kemudian dieksekusi oleh worker untuk menghapus dokumen portofolio terkait di MongoDB; kelemahan cara ini adalah timbulnya latensi konsistensi sementara (*eventual consistency delay*) di mana data portofolio masih sempat terbaca beberapa saat sebelum antrean pesan dieksekusi. Cara penanganan kedua adalah **Validasi Read-Time (*Just-in-Time Verification*)**: setiap kali endpoint portofolio diakses, sistem memverifikasi keberadaan mahasiswa di PostgreSQL dan otomatis menghapus dokumen portofolio di MongoDB saat PostgreSQL mengembalikan status 404; kelemahan cara ini adalah memberikan beban *overhead* I/O kueri tambahan ke PostgreSQL untuk setiap pembacaan portofolio dan dokumen yatim tidak akan pernah terhapus jika endpoint portofolio mahasiswa tersebut tidak pernah dipanggil lagi.

### 3.3 Analisis Ketersediaan Sistem saat Database Mengalami Kegagalan
Apabila basis data MongoDB mengalami gangguan atau mati total (*downtime*), endpoint yang tetap berjalan normal adalah seluruh endpoint master mahasiswa di PostgreSQL (seperti `GET /students`, `POST /students`, `PUT /students`, serta fitur login `POST /auth/login`), sementara seluruh endpoint portofolio (`/students/:id/portfolio` dan `/portfolios/stats/technologies`) serta endpoint health check (`GET /health`) akan gagal dengan pesan status *503 Service Unavailable*. Perilaku isolasi kegagalan (*fault isolation*) ini adalah perilaku yang sangat kami inginkan dalam arsitektur terdistribusi, karena kegagalan pada layanan penunjang sekunder (portofolio) tidak boleh melumpuhkan fungsi bisnis inti universitas (data kemahasiswaan dan otentikasi login), sehingga mahasiswa tetap dapat melakukan administrasi pokok meskipun fitur portofolio sedang mengalami pemeliharaan.

---

## BAB IV
## PENUTUP

### 4.1 Kesimpulan
Berdasarkan seluruh rangkaian implementasi dan pengujian yang telah dilaksanakan pada Modul 10, dapat disimpulkan bahwa arsitektur *Polyglot Persistence* menggunakan kombinasi Go Fiber, PostgreSQL, dan MongoDB berhasil dibangun dengan baik. MongoDB terbukti sangat adaptif dalam menangani struktur data portofolio mahasiswa yang bersarang melalui dokumen BSON. Seluruh kriteria tugas mandiri telah terpenuhi 100%, meliputi:
1. Pembuatan indeks unik yang berhasil membatasi 1 mahasiswa tepat 1 dokumen portofolio, terverifikasi melalui visualisasi `EXPRESS_IXSCAN` pada MongoDB Compass.
2. Keberhasilan implementasi pembaruan atomik (`$set`, `$push`, `$pull`) yang terbukti kebal dari *Race Condition* melalui pengujian 10 request serentak pada kapasitas 20 proyek.
3. Keberhasilan implementasi *Aggregation Pipeline* dengan 2x `$unwind` untuk menyajikan analisis peringkat teknologi terpopuler.
4. Terciptanya sistem pertahanan otomatis terhadap ancaman *NoSQL Injection* berkat implementasi struct Go yang bertipe tetap.

### 4.2 Saran
1. **Penerapan Change Streams:** Ke depan, sistem dapat memanfaatkan fitur *MongoDB Change Streams* untuk mendeteksi pembaruan portofolio secara *real-time* dan menyinkronkannya dengan mesin pencari seperti Elasticsearch.
2. **Implementasi Transaction Multi-Document:** Untuk operasi lintas koleksi di MongoDB yang lebih kompleks pada masa depan, dapat dipertimbangkan pemanfaatan transaksi terdistribusi (*Replica Set Transactions*) guna memastikan kepatuhan ACID penuh di sisi MongoDB.

---
*(Selesai)*

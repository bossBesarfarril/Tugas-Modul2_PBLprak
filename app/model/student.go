package model

import "time"

// Student adalah entitas utama
type Student struct {
	ID        int       `json:"id"`
	NIM       string    `json:"nim"`
	Name      string    `json:"name"`
	Grade     float64   `json:"grade"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	OwnerID   *int      `json:"owner_id,omitempty"`
}

// POST — semua field wajib
type CreateStudentRequest struct {
	NIM   string  `json:"nim"   validate:"required,nim"`
	Name  string  `json:"name"  validate:"required,min=3,max=50"`
	Grade float64 `json:"grade" validate:"required,min=0,max=100"`
}

type ReplaceStudentRequest struct {
	NIM      string  `json:"nim"       validate:"required,nim"`
	Name     string  `json:"name"      validate:"required,min=3,max=50"`
	Grade    float64 `json:"grade"     validate:"required,min=0,max=100"`
	IsActive bool    `json:"is_active"`
}

// Pada PATCH, pointer membedakan "tidak dikirim" (nil) dari "dikirim kosong".
// "omitnil" dipilih karena ia melewati field yang nil (nggak dikirim), 
// tapi bakal TETAP ngecek field yang dikirim string kosong "".
type PatchStudentRequest struct {
	NIM      *string  `json:"nim,omitempty"       validate:"omitnil,nim"`
	Name     *string  `json:"name,omitempty"      validate:"omitnil,min=3,max=50"`
	Grade    *float64 `json:"grade,omitempty"     validate:"omitnil,min=0,max=100"`
	IsActive *bool    `json:"is_active,omitempty"`
}

// Amplop baku untuk semua respons
type WebResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    *Meta  `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type ListQuery struct {
	Page     int
	Limit    int
	Search   string
	Sort     string
	Order    string
	IsActive *bool
}

// Offset menghitung berapa baris yang dilewati untuk halaman ini.
func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}

// Prestasi Struct
type Prestasi struct {
	ID           int    `json:"id"`
	IDMahasiswa  int    `json:"id_mahasiswa"`
	NamaPrestasi string `json:"nama_prestasi"`
	Juara        string `json:"juara"`
}

// ErrorResponse adalah format baku untuk seluruh response kegagalan.
type ErrorResponse struct {
	Success   bool              `json:"success"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

package service

import (
	"testing"

	"api-students/app/model"
)

// TestCountTotalPages menguji rumus pembulatan halaman paginasi.
func TestCountTotalPages(t *testing.T) {
	cases := []struct{ total, limit, want int }{
		{0, 10, 0},
		{1, 10, 1},
		{10, 10, 1},
		{11, 10, 2},
		{137, 20, 7},
	}

	for _, tc := range cases {
		if got := CountTotalPages(tc.total, tc.limit); got != tc.want {
			t.Errorf("total=%d limit=%d: harap %d, dapat %d",
				tc.total, tc.limit, tc.want, got)
		}
	}
}

// TestValidateCreate menguji validasi request POST mahasiswa baru.
func TestValidateCreate(t *testing.T) {
	// Kasus 1: data kosong semua (harus menghasilkan error pada nim, name, grade)
	invalidReq := model.CreateStudentRequest{
		NIM:   "",
		Name:  "",
		Grade: 150, // di luar rentang 0-100
	}
	errs := ValidateCreate(invalidReq)
	if len(errs) != 3 {
		t.Errorf("harap 3 error validasi, dapat %d: %v", len(errs), errs)
	}

	// Kasus 2: data valid (tidak boleh ada error)
	validReq := model.CreateStudentRequest{
		NIM:   "2024001",
		Name:  "Budi Santoso",
		Grade: 85.5,
	}
	errsValid := ValidateCreate(validReq)
	if len(errsValid) != 0 {
		t.Errorf("data valid tidak boleh menghasilkan error, dapat: %v", errsValid)
	}
}

// TestApplyPatch menguji perubahan sebagian field tanpa merusak data lainnya.
func TestApplyPatch(t *testing.T) {
	initial := model.Student{
		ID:       1,
		NIM:      "2024001",
		Name:     "Budi Santoso",
		Grade:    80.0,
		IsActive: true,
	}

	newGrade := 95.5
	inactive := false

	// Hanya mengubah grade dan is_active (NIM dan Name tidak dikirim)
	result, errs := ApplyPatch(initial, model.PatchStudentRequest{
		Grade:    &newGrade,
		IsActive: &inactive,
	})

	if len(errs) != 0 {
		t.Fatalf("tidak seharusnya ada error: %v", errs)
	}
	if result.Grade != 95.5 {
		t.Errorf("grade seharusnya berubah jadi 95.5, dapat %.2f", result.Grade)
	}
	if result.IsActive != false {
		t.Error("is_active seharusnya berubah menjadi false")
	}
	if result.Name != "Budi Santoso" || result.NIM != "2024001" {
		t.Error("field yang tidak dikirim seharusnya tidak berubah")
	}
}

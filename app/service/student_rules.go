package service

import (
	"strings"

	"api-students/app/model"
)

// ValidateCreate memeriksa isian pada permintaan POST mahasiswa.
func ValidateCreate(req model.CreateStudentRequest) map[string]string {
	errs := map[string]string{}

	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "wajib diisi"
	}
	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "wajib diisi"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "harus antara 0 dan 100"
	}

	return errs
}

// ValidateReplace memeriksa isian pada permintaan PUT mahasiswa.
func ValidateReplace(req model.ReplaceStudentRequest) map[string]string {
	errs := map[string]string{}

	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "wajib diisi pada PUT"
	}
	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "wajib diisi pada PUT"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "harus antara 0 dan 100"
	}

	return errs
}

// ApplyPatch menerapkan perubahan sebagian dan memvalidasi nilai barunya.
func ApplyPatch(
	current model.Student, req model.PatchStudentRequest,
) (model.Student, map[string]string) {
	errs := map[string]string{}

	if req.NIM != nil {
		nim := strings.TrimSpace(*req.NIM)
		if nim == "" {
			errs["nim"] = "tidak boleh kosong"
		} else {
			current.NIM = nim
		}
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			errs["name"] = "tidak boleh kosong"
		} else {
			current.Name = name
		}
	}

	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 100 {
			errs["grade"] = "harus antara 0 dan 100"
		} else {
			current.Grade = *req.Grade
		}
	}

	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	return current, errs
}

// IsEmptyPatch menandai permintaan PATCH yang tidak mengirim field apa pun.
func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil
}

// CountTotalPages membulatkan ke atas tanpa memakai bilangan pecahan.
func CountTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}
	return (total + limit - 1) / limit
}
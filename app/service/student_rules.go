package service

import (
	"api-students/app/model"
)





// ApplyPatch hanya bertugas menggabungkan data yang dikirim dengan data lama.
func ApplyPatch(current model.Student, req model.PatchStudentRequest) model.Student {
	if req.NIM != nil {
		current.NIM = *req.NIM
	}
	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.Grade != nil {
		current.Grade = *req.Grade
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}

	return current
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
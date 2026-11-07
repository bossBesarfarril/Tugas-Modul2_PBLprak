package service

import (
	"errors"
	"fmt"
	"testing"

	"api-students/app/repository"
	"api-students/helper"
)

// Catatan Perbaikan Puzzle 3:
// Pada modul aslinya tertulis package service_test dan memanggil service.translateReviewError.
// Gejala: Compiler error "cannot refer to unexported name service.translateReviewError".
// Akar masalah: translateReviewError adalah fungsi private (tidak di-export). Package service_test
// adalah black-box test sehingga tidak bisa mengakses fungsi private.
// Perbaikan: Ubah package menjadi package service (white-box test) dan panggil translateReviewError secara langsung.

func TestTranslateReviewError(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{repository.ErrInvalidID, 400, helper.CodeBadRequest},
		{repository.ErrNotFound, 404, helper.CodeNotFound},
		{repository.ErrDuplicate, 409, "ALREADY_REVIEWED"},
		{repository.ErrNotOwner, 403, helper.CodeForbidden},
		{repository.ErrOwnReview, 409, "OWN_REVIEW"},
		{repository.ErrAlreadyVoted, 409, "ALREADY_VOTED"},
		{repository.ErrReplyLimit, 409, "REPLY_LIMIT"},
		{fmt.Errorf("lapisan luar: %w", repository.ErrDuplicate), 409, "ALREADY_REVIEWED"},
		{errors.New("koneksi MongoDB putus"), 500, helper.CodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			var appErr *helper.AppError
			if !errors.As(translateReviewError(tt.err), &appErr) {
				t.Fatalf("hasil bukan *helper.AppError")
			}
			if appErr.Status != tt.wantStatus || appErr.Code != tt.wantCode {
				t.Errorf("dapat %d %s, seharusnya %d %s", appErr.Status, appErr.Code, tt.wantStatus, tt.wantCode)
			}
		})
	}
}

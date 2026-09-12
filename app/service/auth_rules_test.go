package service

import (
	"testing"

	"api-students/app/model"
)

// TestCheckPasswordStrength menguji validasi kekuatan password
func TestCheckPasswordStrength(t *testing.T) {
	cases := []struct {
		name     string
		password string
		wantErr  string
	}{
		{"terlalu pendek", "abc1", "minimal 8 karakter"},
		{"hanya huruf", "abcdefgh", "harus memuat huruf dan angka"},
		{"hanya angka", "123456789", "harus memuat huruf dan angka"},
		{"password umum", "password1", "password terlalu umum"},
		{"password valid", "rahasia123", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CheckPasswordStrength(tc.password)
			if got != tc.wantErr {
				t.Errorf("password %q: harap %q, dapat %q", tc.password, tc.wantErr, got)
			}
		})
	}
}

// TestValidateRegister menguji validasi pendaftaran user
func TestValidateRegister(t *testing.T) {
	t.Run("input valid", func(t *testing.T) {
		req := model.RegisterRequest{
			Username: "farril_n",
			Email:    "farril@example.com",
			Password: "securePassword123",
		}
		errs := ValidateRegister(req)
		if len(errs) != 0 {
			t.Errorf("harap tidak ada error, dapat: %v", errs)
		}
	})

	t.Run("input tidak valid", func(t *testing.T) {
		req := model.RegisterRequest{
			Username: "a!", // karakter ilegal & pendek
			Email:    "bukan-email",
			Password: "123",
		}
		errs := ValidateRegister(req)
		if len(errs) != 3 {
			t.Errorf("harap 3 error (username, email, password), dapat %d: %v", len(errs), errs)
		}
	})
}

// TestValidateLogin menguji kelengkapan isian login
func TestValidateLogin(t *testing.T) {
	t.Run("field kosong", func(t *testing.T) {
		req := model.LoginRequest{Username: "", Password: ""}
		errs := ValidateLogin(req)
		if len(errs) != 2 {
			t.Errorf("harap 2 error, dapat %d", len(errs))
		}
	})
}

package helper

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword mengubah password menjadi hash bcrypt.
// Nilai asli password tidak pernah disimpan di mana pun.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword membandingkan hash di database dengan password yang diketik.
// Mengembalikan true kalau cocok.
func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// VerifyDummyPassword menjalankan hash palsu agar waktu tanggapnya mirip
// dengan kasus password salah. Tujuannya supaya penyerang ga bisa bedakan
// "username ga ada" vs "password salah" dari kecepatan respons.
func VerifyDummyPassword(password string) {
	dummy := "$2a$12$000000000000000000000uVHaf5FCJuDKKhYsOPe6PUpiUq3bLiOy"
	bcrypt.CompareHashAndPassword([]byte(dummy), []byte(password))
}

// GenerateRandomToken membuat string acak sepanjang n byte,
// dikembalikan dalam bentuk hex string.
func GenerateRandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SHA256Hex menghitung SHA-256 dari sebuah string.
// Dipakai untuk meng-hash refresh token sebelum disimpan ke database.
func SHA256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
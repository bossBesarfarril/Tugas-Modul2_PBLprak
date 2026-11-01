//go:build ignore

package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"

	"api-students/app/model"
	"api-students/config"
	"api-students/database"
	"api-students/helper"
)

func main() {
	config.LoadEnv()
	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer pool.Close()

	hashPassword := func(pwd string) string {
		h, _ := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		return string(h)
	}

	testUsers := []struct {
		username string
		email    string
		password string
		role     string
	}{
		{"admin", "admin@unair.ac.id", "admin123", "admin"},
		{"mhsuji", "mhsuji@student.unair.ac.id", "password123", "mahasiswa"},
		{"salahalamat", "salah@kampus.invalid", "password123", "mahasiswa"},
	}

	for _, u := range testUsers {
		var id int
		err := pool.QueryRow(ctx, "SELECT id FROM users WHERE LOWER(username) = LOWER($1)", u.username).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) || err != nil {
			// Insert baru
			_, err = pool.Exec(ctx, `
				INSERT INTO users (username, email, password, role)
				VALUES ($1, $2, $3, $4)`,
				u.username, u.email, hashPassword(u.password), u.role)
			if err != nil {
				log.Fatalf("Failed to insert user %s: %v", u.username, err)
			}
		} else {
			// Update yang sudah ada
			_, err = pool.Exec(ctx, `
				UPDATE users SET email = $1, password = $2, role = $3 WHERE id = $4`,
				u.email, hashPassword(u.password), u.role, id)
			if err != nil {
				log.Fatalf("Failed to update user %s: %v", u.username, err)
			}
		}
	}

	jwtManager := helper.NewJWTManager(
		config.GetEnv("JWT_SECRET", "super_secret_jwt_key_farril_nasywa_434241124_backend"),
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		24*time.Hour,
	)

	fmt.Println("=== TOKENS FOR TESTING ===")
	for _, u := range testUsers {
		var id int
		err := pool.QueryRow(ctx, "SELECT id FROM users WHERE LOWER(username) = LOWER($1)", u.username).Scan(&id)
		if err != nil {
			log.Fatalf("Get id for %s failed: %v", u.username, err)
		}
		token, err := jwtManager.GenerateAccess(model.User{
			ID:       id,
			Username: u.username,
			Email:    u.email,
			Role:     u.role,
		})
		if err != nil {
			log.Fatalf("Generate token for %s: %v", u.username, err)
		}
		fmt.Printf("USER: %s (ID: %d, Email: %s)\nTOKEN: %s\n\n", u.username, id, u.email, token)
	}
}

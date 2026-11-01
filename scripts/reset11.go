//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"api-students/config"
	"api-students/database"
)

func main() {
	config.LoadEnv()
	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		log.Fatalf("DB connect failed: %v", err)
	}
	defer pool.Close()

	content, err := os.ReadFile("reset11.sql")
	if err != nil {
		log.Fatalf("Read reset11.sql failed: %v", err)
	}

	_, err = pool.Exec(ctx, string(content))
	if err != nil {
		log.Fatalf("Exec reset11.sql failed: %v", err)
	}
	fmt.Println("reset11.sql successfully executed")
}

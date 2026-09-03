package config

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/service"
	"api-students/helper"
	"api-students/middleware"
	"api-students/route"
)

// NewApp merakit aplikasi: membuat instance Fiber, memasang middleware, lalu mendaftarkan route.
func NewApp(
	logger *slog.Logger, pool *pgxpool.Pool, studentService *service.StudentService,
) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      GetEnv("APP_NAME", "API Students - Tugas Mandiri Modul 4"),
		ErrorHandler: newErrorHandler(logger),
	})

	middleware.Register(app, logger)
	route.Register(app, pool, studentService)

	// Penampung terakhir untuk URL yang tidak dikenal (404)
	app.Use(func(c *fiber.Ctx) error {
		return helper.Fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	return app
}

// newErrorHandler menangkap error yang tidak tertangani di service secara konsisten.
func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		message := "terjadi error pada server"

		if e, ok := err.(*fiber.Error); ok {
			status = e.Code
			message = e.Message
		}

		logger.Error("unhandled_error",
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.String("error", err.Error()),
		)

		return helper.Fail(c, status, message)
	}
}

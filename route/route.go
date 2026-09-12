package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"api-students/app/service"
	"api-students/helper"
	"api-students/middleware"
)

// Dependencies mengumpulkan seluruh dependensi yang dibutuhkan oleh routing.
type Dependencies struct {
	Pool           *pgxpool.Pool
	JWT            *helper.JWTManager
	StudentService *service.StudentService
	AuthService    *service.AuthService
}

// Register memetakan URL ke handler service masing-masing.
func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	// --- 1. Endpoint Publik ---
	api.Get("/health", healthCheck(deps.Pool))

	// --- 2. Endpoint Autentikasi ---
	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", deps.AuthService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", deps.AuthService.Refresh)
	auth.Post("/logout", deps.AuthService.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	// --- 3. Endpoint Mahasiswa (TERKUNCI: Wajib Access Token) ---
	students := api.Group("/students",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT),
	)
	students.Get("/", deps.StudentService.List)
	students.Get("/:id", deps.StudentService.Get)
	students.Get("/:id/prestasi", deps.StudentService.GetPrestasi)
	students.Post("/", deps.StudentService.Create)
	students.Put("/:id", deps.StudentService.Replace)
	students.Patch("/:id", deps.StudentService.Patch)
	students.Delete("/:id", deps.StudentService.Delete)
}

// healthCheck melaporkan kondisi layanan web server beserta koneksi database-nya.
func healthCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.Fail(c, fiber.StatusServiceUnavailable, "database tidak dapat dihubungi")
		}

		return helper.Success(c, fiber.StatusOK, "server dan database berjalan", nil)
	}
}

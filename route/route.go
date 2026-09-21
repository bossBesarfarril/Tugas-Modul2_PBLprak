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

type Dependencies struct {
	Pool           *pgxpool.Pool
	JWT            *helper.JWTManager
	Perms          *helper.PermissionSet // <-- Tambahan wadah hak akses
	StudentService *service.StudentService
	AuthService    *service.AuthService
}

func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")
	api.Get("/health", healthCheck(deps.Pool))

	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", deps.AuthService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", deps.AuthService.Refresh)
	auth.Post("/logout", deps.AuthService.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	// Rute mahasiswa wajib bawa token JWT
	students := api.Group("/students",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT),
	)

	// Pemasangan satpam Level 1 (RequirePermission) sesuai aksi
	students.Get("/", middleware.RequirePermission(deps.Perms, "student:list"), deps.StudentService.List)
	students.Get("/:id", middleware.RequirePermission(deps.Perms, "student:read:any"), deps.StudentService.Get)
	students.Get("/:id/prestasi", middleware.RequirePermission(deps.Perms, "student:read:any"), deps.StudentService.GetPrestasi)
	students.Post("/", middleware.RequirePermission(deps.Perms, "student:create"), deps.StudentService.Create)
	students.Put("/:id", middleware.RequirePermission(deps.Perms, "student:update:any"), deps.StudentService.Replace)
	students.Patch("/:id", middleware.RequirePermission(deps.Perms, "student:update:any"), deps.StudentService.Patch)
	students.Delete("/:id", middleware.RequirePermission(deps.Perms, "student:delete"), deps.StudentService.Delete)
}

func healthCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("database error")
		}
		return helper.Success(c, fiber.StatusOK, "server berjalan", nil)
	}
}
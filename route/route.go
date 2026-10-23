package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"api-students/app/service"
	"api-students/helper"
	"api-students/middleware"
)

type Dependencies struct {
	Pool             *pgxpool.Pool
	MongoClient      *mongo.Client
	JWT              *helper.JWTManager
	Perms            *helper.PermissionSet // <-- Tambahan wadah hak akses
	StudentService   *service.StudentService
	AuthService      *service.AuthService
	PortfolioService *service.PortfolioService
}

func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")
	api.Get("/health", healthCheck(deps.Pool, deps.MongoClient))
	api.Get("/portfolios/stats/technologies", deps.PortfolioService.GetTopTechnologies)

	// --- TARUH DI SINI: Portofolio Bebas Akses (Modul 10) ---
	api.Get("/students/:id/portfolio", deps.PortfolioService.GetPortfolio)
	api.Put("/students/:id/portfolio", middleware.RequireJSON, deps.PortfolioService.UpsertPortfolio)
	api.Post("/students/:id/portfolio/projects", middleware.RequireJSON, deps.PortfolioService.AddProject)
	api.Delete("/students/:id/portfolio/projects/:projectId", deps.PortfolioService.DeleteProject)

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

func healthCheck(pool *pgxpool.Pool, mongoClient *mongo.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("PostgreSQL tidak dapat dihubungi")
		}
		if err := mongoClient.Ping(ctx, nil); err != nil {
			return helper.ServiceUnavailable("MongoDB tidak dapat dihubungi")
		}
		return helper.Success(c, fiber.StatusOK, "server, PostgreSQL, dan MongoDB berjalan", nil)
	}
}

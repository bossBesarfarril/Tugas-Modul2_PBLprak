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
	Pool                *pgxpool.Pool
	MongoClient         *mongo.Client
	JWT                 *helper.JWTManager
	Perms               *helper.PermissionSet
	StudentService      *service.StudentService
	AuthService         *service.AuthService
	PortfolioService    *service.PortfolioService
	CourseService       *service.CourseService
	NotificationService *service.NotificationService
}

func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")
	api.Get("/health", healthCheck(deps.Pool, deps.MongoClient))
	api.Get("/portfolios/stats/technologies", deps.PortfolioService.GetTopTechnologies)

	// --- Portofolio (Modul 10) ---
	api.Get("/students/:id/portfolio", deps.PortfolioService.GetPortfolio)
	api.Put("/students/:id/portfolio", middleware.RequireJSON, deps.PortfolioService.UpsertPortfolio)
	api.Post("/students/:id/portfolio/projects", middleware.RequireJSON, deps.PortfolioService.AddProject)
	api.Delete("/students/:id/portfolio/projects/:projectId", deps.PortfolioService.DeleteProject)

	// --- Auth ---
	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", deps.AuthService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", deps.AuthService.Refresh)
	auth.Post("/logout", deps.AuthService.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

	// --- Students ---
	students := api.Group("/students",
		middleware.RequireJSON,
		middleware.RequireAuth(deps.JWT),
	)
	students.Get("/", middleware.RequirePermission(deps.Perms, "student:list"), deps.StudentService.List)
	students.Get("/:id", middleware.RequirePermission(deps.Perms, "student:read:any"), deps.StudentService.Get)
	students.Get("/:id/prestasi", middleware.RequirePermission(deps.Perms, "student:read:any"), deps.StudentService.GetPrestasi)
	students.Post("/", middleware.RequirePermission(deps.Perms, "student:create"), deps.StudentService.Create)
	students.Put("/:id", middleware.RequirePermission(deps.Perms, "student:update:any"), deps.StudentService.Replace)
	students.Patch("/:id", middleware.RequirePermission(deps.Perms, "student:update:any"), deps.StudentService.Patch)
	students.Delete("/:id", middleware.RequirePermission(deps.Perms, "student:delete"), deps.StudentService.Delete)

	// --- Courses & KRS (Modul 9 & Modul 11) ---
	registerCourseRoutes := func(router fiber.Router) {
		router.Get("/", deps.CourseService.List)
		router.Get("/mine", deps.CourseService.Mine)
		router.Get("/:id", deps.CourseService.Get)
		router.Post("/:id/enroll", deps.CourseService.Enroll)
		router.Delete("/:id/enroll", deps.CourseService.Drop)
		router.Post("/:id/swap", deps.CourseService.Swap)
		router.Patch("/:id", middleware.RequirePermission(deps.Perms, "course:manage"), deps.CourseService.UpdateQuota)
	}

	coursesGroup := api.Group("/courses", middleware.RequireAuth(deps.JWT))
	registerCourseRoutes(coursesGroup)

	// Juga layani rute langsung /courses untuk kompatibilitas curl panduan
	rootCoursesGroup := app.Group("/courses", middleware.RequireAuth(deps.JWT))
	registerCourseRoutes(rootCoursesGroup)

	// --- Notifikasi (Modul 11) ---
	registerNotificationRoutes := func(router fiber.Router) {
		router.Get("/mine", deps.NotificationService.Mine)
	}
	notifGroup := api.Group("/notifications", middleware.RequireAuth(deps.JWT))
	registerNotificationRoutes(notifGroup)

	rootNotifGroup := app.Group("/notifications", middleware.RequireAuth(deps.JWT))
	registerNotificationRoutes(rootNotifGroup)

	// --- Admin & Monitoring Queue (Modul 11) ---
	registerAdminRoutes := func(router fiber.Router) {
		router.Get("/queue", middleware.RequirePermission(deps.Perms, "course:manage"), deps.NotificationService.Monitor)
	}
	adminGroup := api.Group("/admin", middleware.RequireAuth(deps.JWT))
	registerAdminRoutes(adminGroup)

	rootAdminGroup := app.Group("/admin", middleware.RequireAuth(deps.JWT))
	registerAdminRoutes(rootAdminGroup)
}

func healthCheck(pool *pgxpool.Pool, mongoClient *mongo.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.Internal(err)
		}

		if err := mongoClient.Ping(ctx, nil); err != nil {
			return helper.Internal(err)
		}

		return helper.Success(c, fiber.StatusOK, "layanan sehat", fiber.Map{
			"status":   "UP",
			"postgres": "connected",
			"mongodb":  "connected",
		})
	}
}

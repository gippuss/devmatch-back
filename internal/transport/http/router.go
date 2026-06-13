package http

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/gippuss/devmatch-back/internal/auth"
	"github.com/gippuss/devmatch-back/internal/service"
	"github.com/gippuss/devmatch-back/internal/transport/http/handlers"
	"github.com/gippuss/devmatch-back/internal/transport/http/middleware"
)

func NewRouter(
	log *zap.Logger,
	tokenManager *auth.TokenManager,
	authService *service.AuthService,
	userService *service.UserService,
	projectService *service.ProjectService,
	applicationService *service.ApplicationService,
	dictionaryService *service.DictionaryService,
	dbPool *pgxpool.Pool,
	corsOrigins []string,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(
		cors.New(cors.Config{
			AllowOrigins:     corsOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}),
		middleware.Recovery(log),
		middleware.RequestLogger(log),
	)

	healthHandler := handlers.NewHealthHandler(dbPool)
	authHandler := handlers.NewAuthHandler(authService)
	meHandler := handlers.NewMeHandler(userService, applicationService)
	projectHandler := handlers.NewProjectHandler(projectService, applicationService)
	applicationHandler := handlers.NewApplicationHandler(applicationService)
	dictionaryHandler := handlers.NewDictionaryHandler(dictionaryService)
	adminHandler := handlers.NewAdminHandler(projectService, applicationService)

	router.GET("/health", healthHandler.Health)
	router.Static("/uploads", "./uploads")

	api := router.Group("/api/v1")
	{
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
		api.POST("/auth/refresh", authHandler.Refresh)
		api.POST("/auth/logout", authHandler.Logout)

		api.GET("/projects", projectHandler.ListProjects)
		api.GET("/projects/:id", projectHandler.GetProject)
		api.GET("/tags", dictionaryHandler.ListTags)
		api.GET("/tags/system", dictionaryHandler.ListSystemTags)
		api.GET("/skills", dictionaryHandler.ListSkills)
		api.GET("/users/:id", meHandler.GetUserProfile)

		secured := api.Group("", middleware.AuthMiddleware(tokenManager))
		{
			secured.GET("/me", meHandler.GetMe)
			secured.PATCH("/me", meHandler.UpdateMe)
			secured.POST("/me/avatar", meHandler.UploadAvatar)
			secured.DELETE("/me/avatar", meHandler.DeleteAvatar)
			secured.GET("/me/applications", meHandler.ListMyApplications)

			secured.POST("/tags", dictionaryHandler.CreateTag)
			secured.DELETE("/tags/:id", dictionaryHandler.DeleteTag)

			secured.GET("/me/projects", projectHandler.ListMyProjects)
			secured.POST("/projects", projectHandler.CreateProject)
			secured.PATCH("/projects/:id", projectHandler.UpdateProject)
			secured.DELETE("/projects/:id", projectHandler.DeleteProject)
			secured.POST("/projects/:id/roles", projectHandler.AddProjectRole)
			secured.PATCH("/project-roles/:role_id", projectHandler.UpdateProjectRole)
			secured.DELETE("/project-roles/:role_id", projectHandler.DeleteProjectRole)
			secured.GET("/projects/:id/applications", projectHandler.ListProjectApplications)

			secured.POST("/project-roles/:id/applications", applicationHandler.Apply)
			secured.PATCH("/applications/:id/status", applicationHandler.Review)

			secured.POST("/projects/:id/appeal", projectHandler.SubmitAppeal)

			admin := secured.Group("/admin", middleware.AdminMiddleware(userService))
			{
				admin.GET("/projects", adminHandler.ListAllProjects)
				admin.POST("/projects/:id/ban", adminHandler.BanProject)
				admin.POST("/projects/:id/unban", adminHandler.UnbanProject)
				admin.POST("/projects/:id/appeal/review", adminHandler.ReviewAppeal)
				admin.DELETE("/projects/:id", adminHandler.DeleteProject)
				admin.GET("/projects/:id/applications", adminHandler.ListProjectApplications)
			}
		}
	}

	return router
}

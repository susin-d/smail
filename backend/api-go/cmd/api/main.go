package main

import (
	"log"
	"net/http"

	"api-go/internal/api/handlers"
	"api-go/internal/api/middleware"
	"api-go/internal/config"
	"api-go/internal/database"
	"api-go/internal/models"
	"api-go/internal/workers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("smail API starting up...")

	// 1. Load config
	cfg := config.GetSettings()
	log.Printf("Primary domain: %s\n", cfg.PrimaryDomain)

	// 2. Initialize database and redis
	database.Init(cfg)
	defer database.Close()

	// 3. AutoMigrate schemas.
	// Continue startup when migration is incompatible with an existing DB schema.
	err := models.SetupMigrations(database.DB)
	if err != nil {
		log.Printf("Warning: migration skipped due to schema incompatibility: %v", err)
	}

	// 4. Start background mail worker
	go workers.WorkerLoop(cfg)

	// 5. Setup Gin Web Server
	if !cfg.IsDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// ─── Middleware ───
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.CorsOriginList
	if len(cfg.CorsOriginList) == 1 && cfg.CorsOriginList[0] == "*" {
		corsConfig.AllowAllOrigins = true
	}
	corsConfig.AllowCredentials = true
	corsConfig.AddAllowHeaders("Authorization")
	r.Use(cors.New(corsConfig))

	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Next()
	})

	// ─── Handlers ───
	authHandler := &handlers.AuthHandler{Cfg: cfg}
	mailHandler := &handlers.MailHandler{}
	domainHandler := &handlers.DomainHandler{}
	userHandler := &handlers.UserHandler{}

	// Health check
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": "smail API", "version": "1.0.0", "status": "operational"})
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"services": gin.H{
				"api":      "up",
				"database": "configured",
				"redis":    "configured",
			},
		})
	})

	// Auth Group
	authGrp := r.Group("/auth")
	{
		authGrp.POST("/login", authHandler.Login)
		authGrp.POST("/register", authHandler.Register)
	}

	// Mail Group
	mailGrp := r.Group("/mail")
	mailGrp.Use(middleware.RequireAuth(cfg))
	{
		mailGrp.GET("/inbox", mailHandler.GetInbox)
		mailGrp.GET("/folders", mailHandler.GetFolders)
		mailGrp.GET("/:mail_id", mailHandler.GetMail)
		mailGrp.POST("/send", mailHandler.SendMail)
		mailGrp.POST("/:mail_id/action", mailHandler.MailAction)
	}

	// Domains Group
	domainGrp := r.Group("/domains")
	{
		domainGrp.Use(middleware.RequireAuth(cfg))
		domainGrp.GET("", domainHandler.ListDomains)
		domainGrp.GET("/:domain_id/dns", domainHandler.GetDomainDNS)
		
		adminDomainGrp := domainGrp.Group("")
		adminDomainGrp.Use(middleware.RequireAdmin())
		adminDomainGrp.POST("", domainHandler.CreateDomain)
		adminDomainGrp.DELETE("/:domain_id", domainHandler.DeleteDomain)
	}

	// Users Group
	userGrp := r.Group("/users")
	userGrp.Use(middleware.RequireAuth(cfg))
	{
		userGrp.GET("/me", userHandler.GetCurrentUserProfile)
		userGrp.PATCH("/me", userHandler.UpdateCurrentUser)
		
		adminUserGrp := userGrp.Group("")
		adminUserGrp.Use(middleware.RequireAdmin())
		adminUserGrp.GET("", userHandler.ListUsers)
		adminUserGrp.POST("", userHandler.CreateUser)
		adminUserGrp.DELETE("/:user_id", userHandler.DeleteUser)
	}

	log.Println("Server is running on :8000")
	if err := r.Run(":8000"); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
}

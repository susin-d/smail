package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"api-go/internal/api/authutils"
	"api-go/internal/api/handlers"
	"api-go/internal/api/middleware"
	"api-go/internal/config"
	"api-go/internal/database"
	"api-go/internal/models"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type testState struct {
	router  *gin.Engine
	cfg     *config.Settings
	domain  models.Domain
	admin   models.User
	user    models.User
	cleanup func()
}

func setupTestState(t *testing.T) *testState {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	database.DB = db

	if err := models.SetupMigrations(database.DB); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}

	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	database.RedisClient = redis.NewClient(&redis.Options{Addr: mini.Addr()})

	cfg := &config.Settings{
		JWTSecret:     "test_secret_key_that_is_long_enough_12345",
		JWTExpireMins: 1440,
		IsDev:         true,
	}

	domain := models.Domain{Domain: "example.com", IsVerified: true}
	if err := database.DB.Create(&domain).Error; err != nil {
		t.Fatalf("failed to create domain seed: %v", err)
	}

	adminHash, _ := authutils.HashPassword("AdminPass123!")
	userHash, _ := authutils.HashPassword("UserPass123!")

	admin := models.User{
		Email:        "admin@example.com",
		PasswordHash: adminHash,
		DomainID:     domain.ID,
		DisplayName:  "Admin",
		IsAdmin:      true,
		IsActive:     true,
	}
	user := models.User{
		Email:        "user@example.com",
		PasswordHash: userHash,
		DomainID:     domain.ID,
		DisplayName:  "User",
		IsAdmin:      false,
		IsActive:     true,
	}

	if err := database.DB.Create(&admin).Error; err != nil {
		t.Fatalf("failed to create admin seed: %v", err)
	}
	if err := database.DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user seed: %v", err)
	}

	r := gin.New()
	authHandler := &handlers.AuthHandler{Cfg: cfg}
	mailHandler := &handlers.MailHandler{}
	domainHandler := &handlers.DomainHandler{}
	userHandler := &handlers.UserHandler{}

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": "smail API", "version": "1.0.0", "status": "operational"})
	})
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	authGrp := r.Group("/auth")
	authGrp.POST("/login", authHandler.Login)
	authGrp.POST("/register", authHandler.Register)

	mailGrp := r.Group("/mail")
	mailGrp.Use(middleware.RequireAuth(cfg))
	mailGrp.GET("/inbox", mailHandler.GetInbox)
	mailGrp.GET("/folders", mailHandler.GetFolders)
	mailGrp.GET("/:mail_id", mailHandler.GetMail)
	mailGrp.POST("/send", mailHandler.SendMail)
	mailGrp.POST("/:mail_id/action", mailHandler.MailAction)

	domainGrp := r.Group("/domains")
	domainGrp.Use(middleware.RequireAuth(cfg))
	domainGrp.GET("", domainHandler.ListDomains)
	domainGrp.GET("/:domain_id/dns", domainHandler.GetDomainDNS)
	adminDomainGrp := domainGrp.Group("")
	adminDomainGrp.Use(middleware.RequireAdmin())
	adminDomainGrp.POST("", domainHandler.CreateDomain)
	adminDomainGrp.DELETE("/:domain_id", domainHandler.DeleteDomain)

	userGrp := r.Group("/users")
	userGrp.Use(middleware.RequireAuth(cfg))
	userGrp.GET("/me", userHandler.GetCurrentUserProfile)
	userGrp.PATCH("/me", userHandler.UpdateCurrentUser)
	adminUserGrp := userGrp.Group("")
	adminUserGrp.Use(middleware.RequireAdmin())
	adminUserGrp.GET("", userHandler.ListUsers)
	adminUserGrp.POST("", userHandler.CreateUser)
	adminUserGrp.DELETE("/:user_id", userHandler.DeleteUser)

	return &testState{
		router: r,
		cfg:    cfg,
		domain: domain,
		admin:  admin,
		user:   user,
		cleanup: func() {
			_ = database.RedisClient.Close()
			mini.Close()
		},
	}
}

func makeRequest(router http.Handler, method, path, token string, body []byte, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func getToken(t *testing.T, cfg *config.Settings, userID uint, email string) string {
	t.Helper()
	token, _, err := authutils.CreateAccessToken(userID, email, cfg.JWTSecret, cfg.JWTExpireMins)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}
	return token
}

func TestAllEndpoints(t *testing.T) {
	ts := setupTestState(t)
	defer ts.cleanup()

	adminToken := getToken(t, ts.cfg, ts.admin.ID, ts.admin.Email)
	userToken := getToken(t, ts.cfg, ts.user.ID, ts.user.Email)

	// Seed one inbox mail for mailbox endpoints.
	seedMail := models.MailMetadata{
		UserID:    ts.user.ID,
		Sender:    "sender@external.com",
		Recipient: ts.user.Email,
		Subject:   "Welcome",
		Folder:    "INBOX",
		IsRead:    false,
	}
	if err := database.DB.Create(&seedMail).Error; err != nil {
		t.Fatalf("failed to seed mail: %v", err)
	}

	t.Run("GET /", func(t *testing.T) {
		resp := makeRequest(ts.router, http.MethodGet, "/", "", nil, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("GET /health", func(t *testing.T) {
		resp := makeRequest(ts.router, http.MethodGet, "/health", "", nil, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("POST /auth/login", func(t *testing.T) {
		body := []byte(`{"email":"user@example.com","password":"UserPass123!"}`)
		resp := makeRequest(ts.router, http.MethodPost, "/auth/login", "", body, "application/json")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("POST /auth/register", func(t *testing.T) {
		body := []byte(`{"email":"newuser@example.com","password":"UserPass123!","display_name":"New User"}`)
		resp := makeRequest(ts.router, http.MethodPost, "/auth/register", "", body, "application/json")
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("GET /domains", func(t *testing.T) {
		resp := makeRequest(ts.router, http.MethodGet, "/domains", userToken, nil, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	var createdDomain models.Domain
	t.Run("POST /domains", func(t *testing.T) {
		body := []byte(`{"domain":"newdomain.com"}`)
		resp := makeRequest(ts.router, http.MethodPost, "/domains", adminToken, body, "application/json")
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", resp.Code, resp.Body.String())
		}
		if err := database.DB.Where("domain = ?", "newdomain.com").First(&createdDomain).Error; err != nil {
			t.Fatalf("expected created domain in db: %v", err)
		}
	})

	t.Run("GET /domains/{id}/dns", func(t *testing.T) {
		path := "/domains/" + strconv.Itoa(int(createdDomain.ID)) + "/dns"
		resp := makeRequest(ts.router, http.MethodGet, path, userToken, nil, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("DELETE /domains/{id}", func(t *testing.T) {
		path := "/domains/" + strconv.Itoa(int(createdDomain.ID))
		resp := makeRequest(ts.router, http.MethodDelete, path, adminToken, nil, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("GET /users/me", func(t *testing.T) {
		resp := makeRequest(ts.router, http.MethodGet, "/users/me", userToken, nil, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("PATCH /users/me", func(t *testing.T) {
		body := []byte(`{"display_name":"Updated User"}`)
		resp := makeRequest(ts.router, http.MethodPatch, "/users/me", userToken, body, "application/json")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("GET /users", func(t *testing.T) {
		resp := makeRequest(ts.router, http.MethodGet, "/users", adminToken, nil, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	var createdUser models.User
	t.Run("POST /users", func(t *testing.T) {
		payload := fmt.Sprintf(`{"email":"ops@example.com","password":"OpsPass123!","domain_id":%d,"display_name":"Ops","is_admin":false}`,
			ts.domain.ID,
		)
		resp := makeRequest(ts.router, http.MethodPost, "/users", adminToken, []byte(payload), "application/json")
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%s", resp.Code, resp.Body.String())
		}
		if err := database.DB.Where("email = ?", "ops@example.com").First(&createdUser).Error; err != nil {
			t.Fatalf("expected created user in db: %v", err)
		}
	})

	t.Run("DELETE /users/{id}", func(t *testing.T) {
		path := "/users/" + strconv.Itoa(int(createdUser.ID))
		resp := makeRequest(ts.router, http.MethodDelete, path, adminToken, nil, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("GET /mail/folders", func(t *testing.T) {
		resp := makeRequest(ts.router, http.MethodGet, "/mail/folders", userToken, nil, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		if !strings.Contains(resp.Body.String(), "INBOX") {
			t.Fatalf("expected INBOX in response body")
		}
	})

	t.Run("GET /mail/inbox", func(t *testing.T) {
		resp := makeRequest(ts.router, http.MethodGet, "/mail/inbox?folder=INBOX&page=1&per_page=10", userToken, nil, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("GET /mail/{id}", func(t *testing.T) {
		path := "/mail/" + strconv.Itoa(int(seedMail.ID))
		resp := makeRequest(ts.router, http.MethodGet, path, userToken, nil, "")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
		}
		var refreshed models.MailMetadata
		if err := database.DB.First(&refreshed, seedMail.ID).Error; err != nil {
			t.Fatalf("failed to reload seeded mail: %v", err)
		}
		if !refreshed.IsRead {
			t.Fatalf("expected mail to be marked as read")
		}
	})

	t.Run("POST /mail/send", func(t *testing.T) {
		var b bytes.Buffer
		w := multipart.NewWriter(&b)
		_ = w.WriteField("to", "dest@example.com")
		_ = w.WriteField("subject", "Test")
		_ = w.WriteField("body", "Hello")
		if err := w.Close(); err != nil {
			t.Fatalf("failed to close multipart writer: %v", err)
		}

		resp := makeRequest(ts.router, http.MethodPost, "/mail/send", userToken, b.Bytes(), w.FormDataContentType())
		if resp.Code != http.StatusAccepted {
			t.Fatalf("expected 202, got %d body=%s", resp.Code, resp.Body.String())
		}
	})

	t.Run("POST /mail/{id}/action", func(t *testing.T) {
		path := "/mail/" + strconv.Itoa(int(seedMail.ID)) + "/action"
		resp := makeRequest(ts.router, http.MethodPost, path, userToken, []byte(`{"action":"star"}`), "application/json")
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
		}

		var refreshed models.MailMetadata
		if err := database.DB.First(&refreshed, seedMail.ID).Error; err != nil {
			t.Fatalf("failed to reload mail after action: %v", err)
		}
		if !refreshed.IsStarred {
			t.Fatalf("expected mail to be starred")
		}
	})

	// Ensure JSON responses are valid for selected endpoints.
	for _, path := range []string{"/", "/health", "/users/me"} {
		resp := makeRequest(ts.router, http.MethodGet, path, userToken, nil, "")
		if resp.Code != http.StatusOK {
			continue
		}
		var out map[string]interface{}
		if err := json.Unmarshal(resp.Body.Bytes(), &out); err != nil {
			t.Fatalf("invalid json at %s: %v", path, err)
		}
	}
}

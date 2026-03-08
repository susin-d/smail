package handlers

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"api-go/internal/api/authutils"
	"api-go/internal/config"
	"api-go/internal/database"
	"api-go/internal/models"
)

type AuthHandler struct {
	Cfg *config.Settings
}

const (
	MaxLoginAttemptsPerIP = 10
	LoginWindowSeconds    = 300
	MaxAccountFailures    = 10
	LockoutMessage        = "Account temporarily locked due to too many failed login attempts. Try again later or contact an admin."
)

type ipRateLimitRecord struct {
	Count        int
	FirstAttempt time.Time
}

var (
	loginAttempts = make(map[string]*ipRateLimitRecord)
	loginMu       sync.Mutex
)

func checkIPRateLimit(ip string) bool {
	loginMu.Lock()
	defer loginMu.Unlock()

	now := time.Now()
	record, exists := loginAttempts[ip]

	if exists {
		elapsed := now.Sub(record.FirstAttempt).Seconds()
		if elapsed > LoginWindowSeconds {
			loginAttempts[ip] = &ipRateLimitRecord{Count: 1, FirstAttempt: now}
		} else if record.Count >= MaxLoginAttemptsPerIP {
			return false
		} else {
			record.Count++
		}
	} else {
		loginAttempts[ip] = &ipRateLimitRecord{Count: 1, FirstAttempt: now}
	}
	return true
}

func recordFailedAttempt(ip string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	if record, exists := loginAttempts[ip]; exists {
		record.Count++
	}
}

// Schemas
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name,omitempty"`
}

type UserResponse struct {
	ID                  uint      `json:"id"`
	Email               string    `json:"email"`
	DisplayName         string    `json:"display_name"`
	DomainID            uint      `json:"domain_id"`
	DomainName          string    `json:"domain_name"`
	IsAdmin             bool      `json:"is_admin"`
	IsActive            bool      `json:"is_active"`
	StorageQuotaMB      int       `json:"storage_quota_mb"`
	StorageUsedMB       int       `json:"storage_used_mb"`
	CreatedAt           time.Time `json:"created_at"`
}

type TokenResponse struct {
	AccessToken string       `json:"access_token"`
	ExpiresIn   int64        `json:"expires_in"` // Unix timestamp
	User        UserResponse `json:"user"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	clientIP := c.ClientIP()
	if !checkIPRateLimit(clientIP) {
		c.JSON(http.StatusTooManyRequests, gin.H{"detail": "Too many login attempts. Try again later."})
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": "Invalid input"})
		return
	}

	var user models.User
	if err := database.DB.Preload("Domain").Where("email = ?", req.Email).First(&user).Error; err != nil {
		recordFailedAttempt(clientIP)
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Invalid email or password"})
		return
	}

	if user.FailedLoginAttempts >= MaxAccountFailures {
		c.JSON(http.StatusForbidden, gin.H{"detail": LockoutMessage})
		return
	}

	if !authutils.VerifyPassword(req.Password, user.PasswordHash) {
		database.DB.Model(&user).UpdateColumn("failed_login_attempts", user.FailedLoginAttempts+1)
		recordFailedAttempt(clientIP)
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Invalid email or password"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"detail": "Account is disabled"})
		return
	}

	if user.FailedLoginAttempts > 0 {
		database.DB.Model(&user).UpdateColumn("failed_login_attempts", 0)
	}

	token, exp, err := authutils.CreateAccessToken(user.ID, user.Email, h.Cfg.JWTSecret, h.Cfg.JWTExpireMins)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Error generating token"})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		AccessToken: token,
		ExpiresIn:   exp,
		User: UserResponse{
			ID:             user.ID,
			Email:          user.Email,
			DisplayName:    user.DisplayName,
			DomainID:       user.DomainID,
			DomainName:     user.Domain.Domain,
			IsAdmin:        user.IsAdmin,
			IsActive:       user.IsActive,
			StorageQuotaMB: user.StorageQuotaMB,
			StorageUsedMB:  user.StorageUsedMB,
			CreatedAt:      user.CreatedAt,
		},
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	clientIP := c.ClientIP()
	if !checkIPRateLimit(clientIP) {
		c.JSON(http.StatusTooManyRequests, gin.H{"detail": "Too many login attempts. Try again later."})
		return
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": "Invalid input"})
		return
	}

	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"detail": "Email already registered"})
		return
	}

	parts := strings.Split(req.Email, "@")
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid email format"})
		return
	}
	emailDomain := parts[1]

	var domain models.Domain
	if err := database.DB.Where("domain = ?", emailDomain).First(&domain).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Domain '" + emailDomain + "' is not registered on this platform"})
		return
	}

	hash, err := authutils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to process password"})
		return
	}

	user := models.User{
		Email:        req.Email,
		PasswordHash: hash,
		DomainID:     domain.ID,
		DisplayName:  req.DisplayName,
		IsAdmin:      false,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to create user"})
		return
	}

	token, exp, err := authutils.CreateAccessToken(user.ID, user.Email, h.Cfg.JWTSecret, h.Cfg.JWTExpireMins)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Error generating token"})
		return
	}

	c.JSON(http.StatusCreated, TokenResponse{
		AccessToken: token,
		ExpiresIn:   exp,
		User: UserResponse{
			ID:             user.ID,
			Email:          user.Email,
			DisplayName:    user.DisplayName,
			DomainID:       user.DomainID,
			DomainName:     domain.Domain,
			IsAdmin:        user.IsAdmin,
			IsActive:       user.IsActive,
			StorageQuotaMB: user.StorageQuotaMB,
			StorageUsedMB:  user.StorageUsedMB,
			CreatedAt:      user.CreatedAt,
		},
	})
}

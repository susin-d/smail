package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"api-go/internal/api/authutils"
	"api-go/internal/api/middleware"
	"api-go/internal/database"
	"api-go/internal/models"
)

type UserHandler struct{}

type CreateUserRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DomainID    uint   `json:"domain_id" binding:"required"`
	DisplayName string `json:"display_name,omitempty"`
	IsAdmin     bool   `json:"is_admin"`
}

type UpdateUserRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	Password    *string `json:"password,omitempty" binding:"omitempty,min=8"`
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	var users []models.User
	database.DB.Preload("Domain").Order("created_at desc").Find(&users)

	response := make([]UserResponse, len(users))
	for i, u := range users {
		response[i] = UserResponse{
			ID:             u.ID,
			Email:          u.Email,
			DisplayName:    u.DisplayName,
			DomainID:       u.DomainID,
			DomainName:     u.Domain.Domain,
			IsAdmin:        u.IsAdmin,
			IsActive:       u.IsActive,
			StorageQuotaMB: u.StorageQuotaMB,
			StorageUsedMB:  u.StorageUsedMB,
			CreatedAt:      u.CreatedAt,
		}
	}
	c.JSON(http.StatusOK, response)
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": "Invalid input"})
		return
	}

	var domain models.Domain
	if err := database.DB.First(&domain, req.DomainID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Domain not found"})
		return
	}

	parts := strings.Split(req.Email, "@")
	if len(parts) != 2 || parts[1] != domain.Domain {
		c.JSON(http.StatusBadRequest, gin.H{"detail": fmt.Sprintf("Email domain must match '%s'", domain.Domain)})
		return
	}

	var existing models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"detail": "Email already registered"})
		return
	}

	hash, _ := authutils.HashPassword(req.Password)
	user := models.User{
		Email:        req.Email,
		PasswordHash: hash,
		DomainID:     req.DomainID,
		DisplayName:  req.DisplayName,
		IsAdmin:      req.IsAdmin,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, UserResponse{
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
	})
}

func (h *UserHandler) GetCurrentUserProfile(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Not authorized"})
		return
	}

	c.JSON(http.StatusOK, UserResponse{
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
	})
}

func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": "Invalid input"})
		return
	}

	user := middleware.GetCurrentUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"detail": "Not authorized"})
		return
	}

	updates := make(map[string]interface{})
	if req.DisplayName != nil {
		updates["display_name"] = *req.DisplayName
		user.DisplayName = *req.DisplayName
	}
	if req.Password != nil {
		hash, _ := authutils.HashPassword(*req.Password)
		updates["password_hash"] = hash
	}

	if len(updates) > 0 {
		database.DB.Model(user).Updates(updates)
	}

	c.JSON(http.StatusOK, UserResponse{
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
	})
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("user_id")
	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "User not found"})
		return
	}

	database.DB.Delete(&user)
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("User '%s' deleted", user.Email)})
}

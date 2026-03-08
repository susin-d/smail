package handlers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"api-go/internal/api/middleware"
	"api-go/internal/database"
	"api-go/internal/models"
	"api-go/internal/services"
	"github.com/gin-gonic/gin"
)

type MailHandler struct{}

type MailSummary struct {
	ID             uint      `json:"id"`
	Sender         string    `json:"sender"`
	Recipient      string    `json:"recipient"`
	Subject        string    `json:"subject"`
	Folder         string    `json:"folder"`
	IsRead         bool      `json:"is_read"`
	IsStarred      bool      `json:"is_starred"`
	HasAttachments bool      `json:"has_attachments"`
	Timestamp      time.Time `json:"timestamp"`
}

type InboxResponse struct {
	Total   int64         `json:"total"`
	Page    int           `json:"page"`
	PerPage int           `json:"per_page"`
	Mails   []MailSummary `json:"mails"`
}

type FolderResponse struct {
	Name   string `json:"name"`
	Count  int64  `json:"count"`
	Unread int64  `json:"unread"`
}

type MailDetail struct {
	ID             uint      `json:"id"`
	Sender         string    `json:"sender"`
	Recipient      string    `json:"recipient"`
	Subject        string    `json:"subject"`
	Folder         string    `json:"folder"`
	IsRead         bool      `json:"is_read"`
	IsStarred      bool      `json:"is_starred"`
	HasAttachments bool      `json:"has_attachments"`
	Body           string    `json:"body"`
	HTMLBody       *string   `json:"html_body,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	// Attachments missing for simplicity in matching py implementation
}

type SendMailRequest struct {
	To       string  `json:"to" binding:"required,email"`
	Subject  string  `json:"subject"`
	Body     string  `json:"body"`
	HTMLBody *string `json:"html_body,omitempty"`
}

type MailActionRequest struct {
	Action string `json:"action" binding:"required"`
	Folder string `json:"folder,omitempty"`
}

func (h *MailHandler) GetInbox(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	folder := c.DefaultQuery("folder", "INBOX")
	pageStr := c.DefaultQuery("page", "1")
	perPageStr := c.DefaultQuery("per_page", "50")

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(perPageStr)
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	offset := (page - 1) * perPage

	var total int64
	database.DB.Model(&models.MailMetadata{}).Where("user_id = ? AND folder = ?", user.ID, folder).Count(&total)

	var mails []models.MailMetadata
	database.DB.Where("user_id = ? AND folder = ?", user.ID, folder).
		Order("timestamp desc").
		Offset(offset).
		Limit(perPage).
		Find(&mails)

	summary := make([]MailSummary, len(mails))
	for i, m := range mails {
		summary[i] = MailSummary{
			ID:             m.ID,
			Sender:         m.Sender,
			Recipient:      m.Recipient,
			Subject:        m.Subject,
			Folder:         m.Folder,
			IsRead:         m.IsRead,
			IsStarred:      m.IsStarred,
			HasAttachments: m.HasAttachments,
			Timestamp:      m.Timestamp,
		}
	}

	c.JSON(http.StatusOK, InboxResponse{
		Total:   total,
		Page:    page,
		PerPage: perPage,
		Mails:   summary,
	})
}

func (h *MailHandler) GetFolders(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	type Result struct {
		Folder string
		Count  int64
		Unread int64
	}
	var results []Result

	// GORM raw query for counts
	database.DB.Model(&models.MailMetadata{}).
		Select("folder, COUNT(id) as count, SUM(CASE WHEN is_read = 0 THEN 1 ELSE 0 END) as unread").
		Where("user_id = ?", user.ID).
		Group("folder").
		Scan(&results)

	defaultFolders := []string{"INBOX", "Sent", "Drafts", "Trash", "Spam"}
	folderMap := make(map[string]FolderResponse)

	for _, name := range defaultFolders {
		folderMap[name] = FolderResponse{Name: name, Count: 0, Unread: 0}
	}

	for _, r := range results {
		folderMap[r.Folder] = FolderResponse{
			Name:   r.Folder,
			Count:  r.Count,
			Unread: r.Unread,
		}
	}

	var response []FolderResponse
	for _, name := range defaultFolders {
		response = append(response, folderMap[name])
	}
	for _, r := range results {
		found := false
		for _, name := range defaultFolders {
			if name == r.Folder {
				found = true
				break
			}
		}
		if !found {
			response = append(response, folderMap[r.Folder])
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *MailHandler) GetMail(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	mailID := c.Param("mail_id")

	var mail models.MailMetadata
	if err := database.DB.Where("id = ? AND user_id = ?", mailID, user.ID).First(&mail).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Email not found"})
		return
	}

	if !mail.IsRead {
		database.DB.Model(&mail).Update("is_read", true)
		mail.IsRead = true
	}

	c.JSON(http.StatusOK, MailDetail{
		ID:             mail.ID,
		Sender:         mail.Sender,
		Recipient:      mail.Recipient,
		Subject:        mail.Subject,
		Folder:         mail.Folder,
		IsRead:         mail.IsRead,
		IsStarred:      mail.IsStarred,
		HasAttachments: mail.HasAttachments,
		Body:           "Email body loaded from mail storage.", // Placeholder
		HTMLBody:       nil,
		Timestamp:      mail.Timestamp,
	})
}

func (h *MailHandler) SendMail(c *gin.Context) {
	user := middleware.GetCurrentUser(c)

	reader, err := c.Request.MultipartReader()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Expected multipart/form-data"})
		return
	}

	var to, subject, body string
	var htmlBodyStr string
	var htmlBody *string
	var attachments []string
	var totalSize int

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"detail": "Error reading form data"})
			return
		}

		formName := part.FormName()
		if part.FileName() == "" {
			buf := new(bytes.Buffer)
			_, _ = io.Copy(buf, part)
			val := buf.String()
			switch formName {
			case "to":
				to = val
			case "subject":
				subject = val
			case "body":
				body = val
			case "html_body":
				htmlBodyStr = val
				htmlBody = &htmlBodyStr
			}
			totalSize += len(val)
		} else {
			fileName := part.FileName()
			_ = os.MkdirAll("/maildata/tmp_attachments", 0777)
			
			// Store with timestamp prefix to prevent collisions, but keep original filename embedded
			tmpFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), fileName)
			tmpPath := filepath.Join("/maildata/tmp_attachments", tmpFileName)
			
			out, err := os.Create(tmpPath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"detail": "Error saving attachment server-side"})
				return
			}
			
			written, err := io.Copy(out, part)
			out.Close()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to stream attachment"})
				return
			}
			attachments = append(attachments, tmpPath)
			totalSize += int(written)
		}
	}

	if to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "'to' field is required"})
		return
	}

	sentMeta := models.MailMetadata{
		UserID:         user.ID,
		Sender:         user.Email,
		Recipient:      to,
		Subject:        subject,
		Folder:         "Sent",
		IsRead:         true,
		HasAttachments: len(attachments) > 0,
		Size:           totalSize,
	}
	database.DB.Create(&sentMeta)

	jobID := services.EnqueueMailJob(user.Email, to, subject, body, htmlBody, attachments, user.ID)

	c.JSON(http.StatusAccepted, gin.H{
		"message": fmt.Sprintf("Email queued for delivery (job: %s)", jobID),
		"success": true,
	})
}

func (h *MailHandler) MailAction(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	mailID := c.Param("mail_id")

	var req MailActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": "Invalid input"})
		return
	}

	var mail models.MailMetadata
	if err := database.DB.Where("id = ? AND user_id = ?", mailID, user.ID).First(&mail).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Email not found"})
		return
	}

	updates := make(map[string]interface{})
	switch req.Action {
	case "read":
		updates["is_read"] = true
	case "unread":
		updates["is_read"] = false
	case "star":
		updates["is_starred"] = true
	case "unstar":
		updates["is_starred"] = false
	case "delete":
		updates["folder"] = "Trash"
	case "move":
		if req.Folder == "" {
			c.JSON(http.StatusBadRequest, gin.H{"detail": "Folder name required for move action"})
			return
		}
		updates["folder"] = req.Folder
	default:
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid action"})
		return
	}

	database.DB.Model(&mail).Updates(updates)

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Action '%s' applied to email %s", req.Action, mailID)})
}

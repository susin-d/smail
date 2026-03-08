package models

import (
	"time"

	"gorm.io/gorm"
)

// Domain represents an email domain
type Domain struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Domain        string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"domain"`
	IsVerified    bool      `gorm:"default:false" json:"is_verified"`
	DKIMPublicKey string    `gorm:"type:text;default:null" json:"dkim_public_key,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	Users []User `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

// User represents an email user (mailbox)
type User struct {
	ID                  uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Email               string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash        string    `gorm:"type:varchar(255);not null" json:"-"`
	DomainID            uint      `gorm:"not null" json:"domain_id"`
	DisplayName         string    `gorm:"type:varchar(100);default:null" json:"display_name,omitempty"`
	IsAdmin             bool      `gorm:"default:false" json:"is_admin"`
	IsActive            bool      `gorm:"default:true" json:"is_active"`
	StorageQuotaMB      int       `gorm:"default:100" json:"storage_quota_mb"`
	StorageUsedMB       int       `gorm:"default:0" json:"storage_used_mb"`
	FailedLoginAttempts int       `gorm:"default:0" json:"failed_login_attempts"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`

	Domain    Domain         `gorm:"foreignKey:DomainID" json:"domain,omitempty"`
	Mails     []MailMetadata `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
	Sessions  []Session      `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

// MailMetadata represents an incoming or outgoing email
type MailMetadata struct {
	ID             uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint      `gorm:"index:idx_user_folder;index:idx_user_timestamp;not null" json:"user_id"`
	MessageID      string    `gorm:"type:varchar(255);index;default:null" json:"message_id,omitempty"`
	Sender         string    `gorm:"type:varchar(255);not null" json:"sender"`
	Recipient      string    `gorm:"type:varchar(255);not null" json:"recipient"`
	Subject        string    `gorm:"type:varchar(500);default:null" json:"subject,omitempty"`
	Folder         string    `gorm:"type:varchar(50);default:'INBOX';index:idx_user_folder" json:"folder"`
	IsRead         bool      `gorm:"default:false" json:"is_read"`
	IsStarred      bool      `gorm:"default:false" json:"is_starred"`
	HasAttachments bool      `gorm:"default:false" json:"has_attachments"`
	Size           int       `gorm:"default:0" json:"size"`
	SpamScore      float64   `gorm:"default:0.0" json:"spam_score"`
	Timestamp      time.Time `gorm:"index:idx_user_timestamp;default:CURRENT_TIMESTAMP" json:"timestamp"`

	User        User         `gorm:"foreignKey:UserID" json:"-"`
	Attachments []Attachment `gorm:"foreignKey:MailID;references:ID;constraint:OnDelete:CASCADE;" json:"attachments,omitempty"`
}

// Session represents an active user login session
type Session struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	TokenHash string    `gorm:"type:varchar(255);index;not null" json:"-"`
	IPAddress string    `gorm:"type:varchar(45);default:null" json:"ip_address,omitempty"`
	UserAgent string    `gorm:"type:varchar(500);default:null" json:"user_agent,omitempty"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

// Attachment represents an email attachment
type Attachment struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	MailID      uint      `gorm:"not null" json:"mail_id"`
	Filename    string    `gorm:"type:varchar(255);not null" json:"filename"`
	MimeType    string    `gorm:"type:varchar(100);default:null" json:"mime_type,omitempty"`
	Size        int       `gorm:"default:0" json:"size"`
	StoragePath string    `gorm:"type:varchar(500);not null" json:"-"`
	CreatedAt   time.Time `json:"created_at"`

	Mail MailMetadata `gorm:"foreignKey:MailID" json:"-"`
}

// SetupMigrations performs auto-migration of the models
func SetupMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&Domain{},
		&User{},
		&MailMetadata{},
		&Session{},
		&Attachment{},
	)
}

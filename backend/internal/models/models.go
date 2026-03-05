package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents an administrator or standard user in the system
type User struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Email           string    `gorm:"uniqueIndex;not null" json:"email"`
	Password        string    `gorm:"not null" json:"-"`
	Name            string    `gorm:"type:varchar(255)" json:"name"`
	Department      string    `gorm:"type:varchar(255)" json:"department"`
	Position        string    `gorm:"type:varchar(255)" json:"position"`
	Phone           string    `gorm:"type:varchar(50)" json:"phone"`
	ProfileImageURL string    `gorm:"type:text" json:"profile_image_url"`
	Role            string    `gorm:"type:varchar(20);default:'user'" json:"role"` // 'admin' or 'user'
	IsApproved      bool      `gorm:"default:false" json:"is_approved"`
	IsBlocked       bool      `gorm:"default:false" json:"is_blocked"`
	Projects        []Project `gorm:"foreignKey:UserID" json:"projects,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Project represents a workspace containing configured APIs
type Project struct {
	ID                   uint                 `gorm:"primaryKey" json:"id"`
	Name                 string               `gorm:"not null" json:"name"`
	Description          string               `json:"description"`
	EnvironmentVariables string               `gorm:"type:text;default:'{}'" json:"environment_variables"`
	UserID               uint                 `gorm:"not null" json:"user_id"`
	APIs                 []API                `gorm:"foreignKey:ProjectID" json:"apis,omitempty"`
	NotificationConfigs  []NotificationConfig `gorm:"foreignKey:ProjectID" json:"notification_configs,omitempty"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
}

// API represents a single API endpoint to be monitored
type API struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	ProjectID          uint           `gorm:"not null" json:"project_id"`
	Folder             string         `gorm:"type:varchar(255);default:'Uncategorized'" json:"folder"`
	Name               string         `gorm:"not null" json:"name"`
	Method             string         `gorm:"not null" json:"method"`      // GET, POST, PUT, DELETE, etc.
	URL                string         `gorm:"not null" json:"url"`         // The full endpoint URL
	Parameters         string         `gorm:"type:text" json:"parameters"` // JSON stringified query params
	Headers            string         `gorm:"type:text" json:"headers"`    // JSON stringified headers
	Body               string         `gorm:"type:text" json:"body"`       // JSON stringified body
	ExpectedStatusCode int            `gorm:"default:200" json:"expected_status_code"`
	Interval           int            `gorm:"default:60" json:"interval"`       // Monitoring interval in seconds
	ScheduleConfig     string         `gorm:"type:text" json:"schedule_config"` // JSON Schedule settings mapping n8n
	OrderIndex         int            `gorm:"default:0" json:"order_index"`
	IsActive           bool           `gorm:"default:true" json:"is_active"`
	Logs               []MonitorLog   `gorm:"foreignKey:ApiID" json:"logs,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"deleted_at"` // Soft delete
}

// MonitorLog represents a health check result for an API
type MonitorLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	ApiID        uint      `gorm:"not null" json:"api_id"`
	StatusCode   int       `json:"status_code"`
	ResponseTime int64     `json:"response_time"` // in milliseconds
	IsSuccess    bool      `json:"is_success"`
	ErrorMessage string    `gorm:"type:text" json:"error_message"`
	ResponseBody string    `gorm:"type:text" json:"response_body"`
	CheckedAt    time.Time `json:"checked_at"`
	API          *API      `gorm:"foreignKey:ApiID" json:"api,omitempty"`
}

// NotificationConfig stores channel preferences for alerting when an API fails
type NotificationConfig struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	ProjectID        uint      `gorm:"index;not null" json:"project_id"`
	EnableTelegram   bool      `gorm:"default:false" json:"enable_telegram"`
	TelegramBotToken string    `json:"telegram_bot_token"`
	TelegramChatID   string    `json:"telegram_chat_id"`
	EnableLINE       bool      `gorm:"default:false" json:"enable_line"`
	LINEUserID       string    `json:"line_user_id"`
	EnableEmail      bool      `gorm:"default:false" json:"enable_email"`
	EmailAddress     string    `json:"email_address"` // Multi-email as comma-separated
	SmtpHost         string    `json:"smtp_host"`
	SmtpPort         int       `json:"smtp_port"`
	SmtpUser         string    `json:"smtp_user"`
	SmtpPass         string    `json:"smtp_pass"`
	EnableTicketing  bool      `gorm:"default:false" json:"enable_ticketing"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

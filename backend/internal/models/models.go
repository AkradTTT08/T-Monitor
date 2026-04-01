package models

import (
	"time"

	"gorm.io/gorm"
)

// Company represents a business entity that groups multiple projects
type Company struct {
	ID          uint            `gorm:"primaryKey" json:"id"`
	Name        string          `gorm:"not null" json:"name"`
	Description string          `json:"description"`
	LogoURL     string          `gorm:"type:text" json:"logo_url"`
	UserID      uint            `gorm:"not null" json:"user_id"`
	Owner       *User           `gorm:"foreignKey:UserID" json:"owner"`
	Projects    []Project       `gorm:"foreignKey:CompanyID" json:"projects"`
	Members     []CompanyMember `gorm:"foreignKey:CompanyID" json:"members"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `gorm:"index" json:"deleted_at"` // Soft delete
}

// CompanyMember links users to companies they have been invited to
type CompanyMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CompanyID uint      `gorm:"not null;index" json:"company_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Role      string    `gorm:"type:varchar(20);default:'member'" json:"role"` // 'owner' or 'member'
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

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
	CoverImageURL        string               `gorm:"type:text" json:"cover_image_url"`
	CoverPosition        int                  `gorm:"default:50" json:"cover_position"`
	UserID               uint                 `gorm:"not null" json:"user_id"`
	CompanyID            *uint                `json:"company_id"`
	APIs                 []API                `gorm:"foreignKey:ProjectID" json:"apis,omitempty"`
	Members              []ProjectMember      `gorm:"foreignKey:ProjectID" json:"members,omitempty"`
	NotificationConfigs  []NotificationConfig `gorm:"foreignKey:ProjectID" json:"notification_configs,omitempty"`
	RepairTasks          []RepairTask         `gorm:"foreignKey:ProjectID" json:"repair_tasks,omitempty"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
	DeletedAt            gorm.DeletedAt       `gorm:"index" json:"deleted_at"` // Soft delete
}

// ProjectMember links users to projects they have been added to
type ProjectMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"not null;index" json:"project_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	Role      string    `gorm:"type:varchar(20);default:'member'" json:"role"` // 'owner' or 'member'
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CreatedAt time.Time `json:"created_at"`
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
	PausedUntil        *time.Time     `json:"paused_until"` // Added for pause feature
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
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"` // Soft delete
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
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deleted_at"` // Soft delete
}

// RepairTask represents a ticket created when an API monitoring check fails
type RepairTask struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	ProjectID    uint           `gorm:"index;not null" json:"project_id"`
	ApiID        uint           `gorm:"not null" json:"api_id"`
	API          *API           `gorm:"foreignKey:ApiID" json:"api,omitempty"`
	Status       string         `gorm:"type:varchar(20);default:'open'" json:"status"` // 'open', 'pending', 'closed', 'failed'
	ErrorMessage string         `gorm:"type:text" json:"error_message"`
	Description  string         `gorm:"type:text" json:"description"`  // For 'failed' status or general notes
	Reason       string         `gorm:"type:text" json:"reason"`       // For 'closed' status
	DocumentURL  string         `gorm:"type:text" json:"document_url"` // Legacy for 'closed' status
	Documents    string         `gorm:"type:text" json:"documents"`    // JSON array of document URLs
	ApprovedBy   *uint          `json:"approved_by"`
	ApprovedAt   *time.Time     `json:"approved_at"`
	ClosedAt     *time.Time     `json:"closed_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// DashboardNotification represents a popup alert for the UI
type DashboardNotification struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserID       uint           `gorm:"index" json:"user_id"` // 0 for system-wide/all admins
	ProjectID    uint           `gorm:"index" json:"project_id"`
	InvitationID *uint          `json:"invitation_id"` // Link to company invitation if type is 'company_invite'
	Type         string         `json:"type"`          // 'api_fail', 'task_approve', 'task_close', 'task_fail', 'company_invite'
	Title        string         `json:"title"`
	Message      string         `json:"message"`
	IsRead       bool           `gorm:"default:false" json:"is_read"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// CompanyInvitation tracks the state of an invitation to join a company
type CompanyInvitation struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CompanyID uint           `gorm:"not null;index" json:"company_id"`
	InviterID uint           `gorm:"not null" json:"inviter_id"`
	InviteeID uint           `gorm:"not null;index" json:"invitee_id"`
	Status    string         `gorm:"type:varchar(20);default:'pending'" json:"status"` // 'pending', 'accepted', 'declined'
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`

	// Preloads
	Company *Company `gorm:"foreignKey:CompanyID" json:"company,omitempty"`
	Inviter *User    `gorm:"foreignKey:InviterID" json:"inviter,omitempty"`
	Invitee *User    `gorm:"foreignKey:InviteeID" json:"invitee,omitempty"`
}

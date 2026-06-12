package workers

import (
	"log"
	"time"

	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

const (
	// LogRetentionDays คือจำนวนวันที่เก็บ MonitorLog ไว้ (ลบ log เก่ากว่านี้)
	LogRetentionDays = 30

	// NotificationRetentionDays คือจำนวนวันที่เก็บ DashboardNotification ที่อ่านแล้ว
	NotificationRetentionDays = 7
)

// StartCleanupWorker เริ่ม background worker ลบ log เก่าทุกวัน
func StartCleanupWorker() {
	// รันครั้งแรกทันทีเมื่อ start เพื่อล้าง backlog
	go runCleanup()

	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		for range ticker.C {
			runCleanup()
		}
	}()
}

func runCleanup() {
	log.Println("[Cleanup] Starting scheduled log cleanup...")

	cleanupMonitorLogs()
	cleanupDashboardNotifications()

	log.Println("[Cleanup] Log cleanup completed.")
}

// cleanupMonitorLogs ลบ MonitorLog ที่เก่ากว่า LogRetentionDays วัน
func cleanupMonitorLogs() {
	cutoff := time.Now().AddDate(0, 0, -LogRetentionDays)

	result := database.DB.
		Where("checked_at < ?", cutoff).
		Delete(&models.MonitorLog{})

	if result.Error != nil {
		log.Printf("[Cleanup] Error deleting old MonitorLogs: %v", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Printf("[Cleanup] Deleted %d MonitorLog records older than %d days", result.RowsAffected, LogRetentionDays)
	}
}

// cleanupDashboardNotifications ลบ DashboardNotification ที่อ่านแล้วและเก่ากว่า NotificationRetentionDays วัน
func cleanupDashboardNotifications() {
	cutoff := time.Now().AddDate(0, 0, -NotificationRetentionDays)

	result := database.DB.
		Where("is_read = ? AND created_at < ?", true, cutoff).
		Delete(&models.DashboardNotification{})

	if result.Error != nil {
		log.Printf("[Cleanup] Error deleting old DashboardNotifications: %v", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		log.Printf("[Cleanup] Deleted %d read DashboardNotification records older than %d days", result.RowsAffected, NotificationRetentionDays)
	}
}

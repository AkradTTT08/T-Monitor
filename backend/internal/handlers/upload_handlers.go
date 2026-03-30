package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofiber/fiber/v2"
)

func UploadMultipleFiles(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Failed to parse multipart form"})
	}

	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No files uploaded"})
	}

	// Get absolute path for upload directory
	uploadDir, err := filepath.Abs("./uploads/repair_docs")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error: path resolution failed"})
	}

	// Create directory if not exists
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create upload directory"})
		}
	}

	var urls []string
	for _, file := range files {
		// Generate unique filename
		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("doc_%d_%s%s", time.Now().UnixNano(), filepath.Base(file.Filename), ext)
		savePath := filepath.Join(uploadDir, filename)

		if err := c.SaveFile(file, savePath); err != nil {
			fmt.Printf("❌ Failed to save file %s: %v\n", file.Filename, err)
			continue
		}

		urls = append(urls, "/uploads/repair_docs/"+filename)
	}

	return c.JSON(fiber.Map{
		"message": "Files uploaded successfully",
		"urls":    urls,
	})
}

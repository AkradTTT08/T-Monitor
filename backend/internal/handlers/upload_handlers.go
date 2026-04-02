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
	cwd, _ := os.Getwd()
	uploadDir := filepath.Join(cwd, "uploads", "repair_docs")
	fmt.Printf("📂 Target Upload Directory: %s\n", uploadDir)

	// Create directory if not exists
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		fmt.Printf("📁 Directory missing, creating: %s\n", uploadDir)
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			fmt.Printf("❌ Failed to create directory: %v\n", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create upload directory"})
		}
	}

	var urls []string
	for _, file := range files {
		// Generate unique filename
		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("doc_%d_%s%s", time.Now().UnixNano(), filepath.Base(file.Filename), ext)
		savePath := filepath.Join(uploadDir, filename)
		fmt.Printf("📄 Saving file to: %s\n", savePath)

		if err := c.SaveFile(file, savePath); err != nil {
			fmt.Printf("❌ Failed to save file %s: %v\n", file.Filename, err)
			continue
		}

		// URL path should always use forward slashes
		urlPath := "/uploads/repair_docs/" + filename
		urls = append(urls, urlPath)
		fmt.Printf("✅ File saved, URL: %s\n", urlPath)
	}

	return c.JSON(fiber.Map{
		"message": "Files uploaded successfully",
		"urls":    urls,
	})
}

func DownloadFile(c *fiber.Ctx) error {
	filePath := c.Query("path")
	if filePath == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Path is required"})
	}

	// Clean the path to prevent directory traversal
	cleanPath := filepath.Clean(filePath)
	
	// Check if path starts with /uploads/
	// If it doesn't, or if it tries to escape the uploads directory, reject it
	cwd, _ := os.Getwd()
	// filePath usually looks like /uploads/repair_docs/xxx.png
	// We want to map it to [cwd]/uploads/repair_docs/xxx.png
	
	// Remove leading slash if present
	if len(cleanPath) > 0 && cleanPath[0] == '/' {
		cleanPath = cleanPath[1:]
	}
	
	fullPath := filepath.Join(cwd, cleanPath)
	
	// Basic security check: ensure the fullPath is still inside the [cwd]/uploads directory
	rel, err := filepath.Rel(filepath.Join(cwd, "uploads"), fullPath)
	if err != nil || (len(rel) >= 2 && rel[:2] == "..") {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Invalid file path"})
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "File not found"})
	}

	return c.Download(fullPath)
}

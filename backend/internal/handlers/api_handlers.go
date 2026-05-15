package handlers

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/models"
	"github.com/monitor-api/backend/internal/services"
)

type APIInput struct {
	ProjectID          uuid.UUID `json:"project_id"`
	Folder             string    `json:"folder"`
	Name               string    `json:"name"`
	Method             string    `json:"method"`
	URL                string    `json:"url"`
	Parameters         string    `json:"parameters"`
	Headers            string    `json:"headers"`
	Body               string    `json:"body"`
	ExpectedStatusCode int       `json:"expected_status_code"`
	Interval           int       `json:"interval"`
	ScheduleConfig     string    `json:"schedule_config"`
	ResponseScript     string    `json:"response_script"`
	RecoveryScript     string    `json:"recovery_script"`
	OrderIndex         int       `json:"order_index"`
}

func CreateAPI(c *fiber.Ctx) error {
	var input APIInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)
	mode := c.Query("mode")

	api := models.API{
		ProjectID:          input.ProjectID,
		Folder:             input.Folder,
		Name:               input.Name,
		Method:             input.Method,
		URL:                input.URL,
		Parameters:         input.Parameters,
		Headers:            input.Headers,
		Body:               input.Body,
		ExpectedStatusCode: input.ExpectedStatusCode,
		Interval:           input.Interval,
		ScheduleConfig:     input.ScheduleConfig,
		ResponseScript:     input.ResponseScript,
		RecoveryScript:     input.RecoveryScript,
		OrderIndex:         input.OrderIndex,
	}

	if c.Locals("is_dry_run") == true {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "DRY_RUN: API validation successful. Data not persisted.",
			"data":    api,
		})
	}

	svc := services.NewAPIService(nil)
	if err := svc.CreateAPI(&api, mode, userID, role); err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Project not found" || err.Error() == "Project not found or unauthorized" {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(api)
}

func ReorderAPIs(c *fiber.Ctx) error {
	projectID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	type ReorderItem struct {
		ID         uuid.UUID `json:"id"`
		Folder     string    `json:"folder"`
		OrderIndex int       `json:"order_index"`
	}

	var items []ReorderItem
	if err := c.BodyParser(&items); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	// Map to the service structure
	svcItems := make([]struct {
		ID         uuid.UUID
		Folder     string
		OrderIndex int
	}, len(items))

	for i, item := range items {
		svcItems[i].ID = item.ID
		svcItems[i].Folder = item.Folder
		svcItems[i].OrderIndex = item.OrderIndex
	}

	svc := services.NewAPIService(nil)
	if err := svc.ReorderAPIs(projectID, svcItems, userID, role); err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Project not found" || err.Error() == "Project not found or unauthorized" {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "APIs reordered successfully"})
}

func GetAPIs(c *fiber.Ctx) error {
	projectID := c.Query("project_id")
	search := c.Query("search")
	pageStr := c.Query("page")
	limitStr := c.Query("limit")

	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	page := 1
	limit := 50
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	svc := services.NewAPIService(nil)
	apis, total, err := svc.GetAPIs(projectID, search, page, limit, userID, role)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Project not found" || err.Error() == "Project not found or unauthorized" {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"data":  apis,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func GetAPI(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	svc := services.NewAPIService(nil)
	api, err := svc.GetAPIByID(id, userID, role)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(api)
}

func UpdateAPI(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	var input APIInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	var body map[string]interface{}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	updateData := make(map[string]interface{})
	if _, ok := body["folder"]; ok {
		updateData["folder"] = input.Folder
	}
	if _, ok := body["name"]; ok {
		updateData["name"] = input.Name
	}
	if _, ok := body["method"]; ok {
		updateData["method"] = input.Method
	}
	if _, ok := body["url"]; ok {
		updateData["url"] = input.URL
	}
	if _, ok := body["parameters"]; ok {
		updateData["parameters"] = input.Parameters
	}
	if _, ok := body["headers"]; ok {
		updateData["headers"] = input.Headers
	}
	if _, ok := body["body"]; ok {
		updateData["body"] = input.Body
	}
	if _, ok := body["expected_status_code"]; ok {
		updateData["expected_status_code"] = input.ExpectedStatusCode
	}
	if _, ok := body["interval"]; ok {
		updateData["interval"] = input.Interval
	}
	if _, ok := body["schedule_config"]; ok {
		updateData["schedule_config"] = input.ScheduleConfig
	}
	if _, ok := body["response_script"]; ok {
		updateData["response_script"] = input.ResponseScript
	}
	if _, ok := body["recovery_script"]; ok {
		updateData["recovery_script"] = input.RecoveryScript
	}

	svc := services.NewAPIService(nil)
	api, err := svc.UpdateAPI(id, updateData, userID, role)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "API not found or unauthorized" {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(api)
}

func DeleteAPI(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	svc := services.NewAPIService(nil)
	if err := svc.DeleteAPI(id, userID, role); err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "API not found or unauthorized" {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "API deleted successfully"})
}

func PauseAPI(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	type PauseInput struct {
		DurationMinutes int `json:"duration_minutes"`
	}

	var input PauseInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	svc := services.NewAPIService(nil)
	if err := svc.PauseAPI(id, input.DurationMinutes, userID, role); err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "API not found or unauthorized" {
			status = fiber.StatusNotFound
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Pause status updated successfully"})
}

func TestAPI(c *fiber.Ctx) error {
	type TestRequest struct {
		Method  string            `json:"method"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
		Body    string            `json:"body"`
	}

	var reqData TestRequest
	if err := c.BodyParser(&reqData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request payload"})
	}

	if reqData.URL == "" || reqData.Method == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "URL and Method are required"})
	}

	var reqBody *bytes.Reader
	if reqData.Body != "" {
		reqBody = bytes.NewReader([]byte(reqData.Body))
	} else {
		reqBody = bytes.NewReader([]byte{})
	}

	httpReq, err := http.NewRequest(strings.ToUpper(reqData.Method), reqData.URL, reqBody)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to construct HTTP request"})
	}

	for key, val := range reqData.Headers {
		httpReq.Header.Set(key, val)
	}

	if httpReq.Header.Get("User-Agent") == "" {
		httpReq.Header.Set("User-Agent", "TTT-Monitor-Engine/1.0")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	startTime := time.Now()
	resp, err := client.Do(httpReq)
	latency := time.Since(startTime).Milliseconds()

	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"error":   "Request failed: " + err.Error(),
			"latency": latency,
		})
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	var jsonResponse interface{}
	if err := json.Unmarshal(bodyBytes, &jsonResponse); err == nil {
		return c.JSON(fiber.Map{
			"status":   resp.StatusCode,
			"latency":  latency,
			"response": jsonResponse,
			"is_json":  true,
		})
	}

	return c.JSON(fiber.Map{
		"status":   resp.StatusCode,
		"latency":  latency,
		"response": bodyString,
		"is_json":  false,
	})
}

func UploadPostmanCollection(c *fiber.Ctx) error {
	projectID := c.Query("project_id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "project_id is required"})
	}

	mode := c.Query("mode")

	file, err := c.FormFile("collection")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid upload file"})
	}

	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot read file"})
	}
	defer f.Close()

	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	svc := services.NewAPIService(nil)
	count, err := svc.ImportPostmanCollection(projectID, mode, f, userID, role)
	if err != nil {
		status := fiber.StatusInternalServerError
		if err.Error() == "Project not found" || err.Error() == "Project not found or unauthorized" {
			status = fiber.StatusNotFound
		} else if err.Error() == "Invalid Postman JSON structure" {
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"message": "Collection imported successfully",
		"count":   count,
	})
}

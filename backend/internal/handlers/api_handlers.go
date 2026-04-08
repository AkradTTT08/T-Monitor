package handlers

import (
	"github.com/google/uuid"

	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
)

type APIInput struct {
	ProjectID          uuid.UUID   `json:"project_id"`
	Folder             string `json:"folder"`
	Name               string `json:"name"`
	Method             string `json:"method"`
	URL                string `json:"url"`
	Parameters         string `json:"parameters"`
	Headers            string `json:"headers"`
	Body               string `json:"body"`
	ExpectedStatusCode int    `json:"expected_status_code"`
	Interval           int    `json:"interval"`
	ScheduleConfig     string      `json:"schedule_config"`
	ResponseScript     string      `json:"response_script"`
	OrderIndex         int         `json:"order_index"`
}

func CreateAPI(c *fiber.Ctx) error {
	var input APIInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)
	mode := c.Query("mode")

	// Verify project ownership
	var project models.Project
	if role == "admin" {
		if err := database.DB.First(&project, "id = ?", input.ProjectID).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
		}
	} else {
		if err := database.DB.Where("id = ? AND user_id = ?", input.ProjectID, userID).First(&project).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found or unauthorized"})
		}
	}

	if mode == "replace" {
		if err := database.DB.Where("project_id = ?", input.ProjectID).Delete(&models.API{}).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to clear existing APIs"})
		}
	}

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
		OrderIndex:         input.OrderIndex,
	}

	if c.Locals("is_dry_run") == true {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"message": "DRY_RUN: API validation successful. Data not persisted.",
			"data":    api,
		})
	}

	if err := database.DB.Create(&api).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to create API endpoint"})
	}

	return c.Status(fiber.StatusCreated).JSON(api)
}

func ReorderAPIs(c *fiber.Ctx) error {
	projectID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	// Verify project ownership
	var project models.Project
	if role == "admin" {
		if err := database.DB.First(&project, "id = ?", projectID).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found"})
		}
	} else {
		if err := database.DB.Where("id = ? AND user_id = ?", projectID, userID).First(&project).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Project not found or unauthorized"})
		}
	}

	type ReorderItem struct {
		ID         uuid.UUID   `json:"id"`
		Folder     string `json:"folder"`
		OrderIndex int    `json:"order_index"`
	}

	var items []ReorderItem
	if err := c.BodyParser(&items); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	tx := database.DB.Begin()

	for _, item := range items {
		if err := tx.Model(&models.API{}).Where("id = ? AND project_id = ?", item.ID, projectID).
			Updates(map[string]interface{}{
				"folder":      item.Folder,
				"order_index": item.OrderIndex,
			}).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to reorder APIs"})
		}
	}

	tx.Commit()

	return c.JSON(fiber.Map{"message": "APIs reordered successfully"})
}

func GetAPIs(c *fiber.Ctx) error {
	projectID := c.Query("project_id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	var apis []models.API

	query := database.DB

	if projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	// Filter by ownership if not Admin
	if role != "admin" {
		query = query.Joins("JOIN projects ON projects.id = apis.project_id").Where("projects.user_id = ?", userID)
	}

	query.Order("folder ASC, order_index ASC").Find(&apis)

	return c.JSON(apis)
}

func UpdateAPI(c *fiber.Ctx) error {
	apiID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	var api models.API
	if err := database.DB.First(&api, "id = ?", apiID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "API not found"})
	}

	var project models.Project
	if err := database.DB.Select("user_id").First(&project, "id = ?", api.ProjectID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Associated project not found"})
	}

	// Verify project ownership
	if role != "admin" && project.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized to edit this API"})
	}

	var input APIInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	// Fetch base model out of our join union to update its fields natively
	var baseAPI models.API
	database.DB.First(&baseAPI, "id = ?", apiID)

	baseAPI.Folder = input.Folder
	baseAPI.Name = input.Name
	baseAPI.Method = input.Method
	baseAPI.URL = input.URL
	baseAPI.Parameters = input.Parameters
	baseAPI.Headers = input.Headers
	baseAPI.Body = input.Body
	baseAPI.ExpectedStatusCode = input.ExpectedStatusCode
	baseAPI.Interval = input.Interval
	baseAPI.ScheduleConfig = input.ScheduleConfig
	baseAPI.ResponseScript = input.ResponseScript
	baseAPI.OrderIndex = input.OrderIndex

	if err := database.DB.Save(&baseAPI).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update API endpoint"})
	}

	return c.JSON(baseAPI)
}

func DeleteAPI(c *fiber.Ctx) error {
	apiID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	var api models.API
	if err := database.DB.First(&api, "id = ?", apiID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "API not found"})
	}

	var project models.Project
	if err := database.DB.Select("user_id").First(&project, "id = ?", api.ProjectID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Associated project not found"})
	}

	// Verify project ownership
	if role != "admin" && project.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized to delete this API"})
	}

	var baseAPI models.API
	database.DB.First(&baseAPI, "id = ?", apiID)
	
	if err := database.DB.Delete(&baseAPI).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to delete API endpoint"})
	}

	return c.JSON(fiber.Map{"message": "API endpoint deleted successfully"})
}

// TestAPI proxies a request to bypass browser CORS for ad-hoc Dashboard testing
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

	// Build the HTTP request
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

	// Apply Headers
	for key, val := range reqData.Headers {
		httpReq.Header.Set(key, val)
	}
	
	// Add default User-Agent if not presented
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

	// Attempt to parse JSON response for better display, fallback to raw string
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

// Function to upload parsing Postman Collection JSON
func UploadPostmanCollection(c *fiber.Ctx) error {
	type PostmanHeader struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	type PostmanRequest struct {
		Method string          `json:"method"`
		Header []PostmanHeader `json:"header"`
		Body   struct {
			Mode string `json:"mode"`
			Raw  string `json:"raw"`
		} `json:"body"`
		URL struct {
			Raw string `json:"raw"`
		} `json:"url"`
	}

	type PostmanItem struct {
		Name    string         `json:"name"`
		Request PostmanRequest `json:"request"`
		Item    []json.RawMessage `json:"item"` // Handle nested folders
	}

	type PostmanVariable struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Type  string `json:"type"`
	}

	type PostmanCollection struct {
		Item     []PostmanItem     `json:"item"`
		Variable []PostmanVariable `json:"variable"`
	}

	projectID := c.Query("project_id")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "project_id is required"})
	}
	
	mode := c.Query("mode")

	// Read attached file
	file, err := c.FormFile("collection")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid upload file"})
	}

	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot read file"})
	}
	defer f.Close()

	var collection PostmanCollection
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&collection); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid Postman JSON structure"})
	}

	var parsedAPIs []models.API

	// Recursive internal parser
	var parseItems func(items []PostmanItem, currentFolder string)
	parseItems = func(items []PostmanItem, currentFolder string) {
		for _, item := range items {
			if len(item.Item) > 0 {
				// Nested folder
				var subItems []PostmanItem
				for _, rawSubItem := range item.Item {
					var subItem PostmanItem
					json.Unmarshal(rawSubItem, &subItem)
					subItems = append(subItems, subItem)
				}
				
				folderName := item.Name
				if currentFolder != "" {
					folderName = currentFolder + "/" + item.Name
				}
				
				parseItems(subItems, folderName)
			} else if item.Request.URL.Raw != "" {
				headers, _ := json.Marshal(item.Request.Header)

				method := item.Request.Method
				if method == "" {
					method = "GET"
				}
				
				folderAssign := currentFolder
				if folderAssign == "" {
					folderAssign = "Uncategorized"
				}

				projectUUID, _ := uuid.Parse(projectID)

				parsedAPIs = append(parsedAPIs, models.API{
					ProjectID:          projectUUID,
					Folder:             folderAssign,
					Name:               item.Name,
					Method:             method,
					URL:                item.Request.URL.Raw,
					Parameters:         "[]",
					Headers:            string(headers),
					Body:               item.Request.Body.Raw,
					ExpectedStatusCode: 200,
					Interval:           60,
				})
			}
		}
	}

	parseItems(collection.Item, "")

	if len(parsedAPIs) > 0 {
		if mode == "replace" {
			if err := database.DB.Unscoped().Where("api_id IN (SELECT id FROM apis WHERE project_id = ?)", projectID).Delete(&models.MonitorLog{}).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to clear existing monitor logs"})
			}

			if err := database.DB.Unscoped().Where("project_id = ?", projectID).Delete(&models.API{}).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to clear existing APIs"})
			}
		}

		if err := database.DB.Create(&parsedAPIs).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save APIs to DB"})
		}
	}

	// Update Project Environment Variables if defined in the collection
	if len(collection.Variable) > 0 {
		var project models.Project
		if err := database.DB.First(&project, "id = ?", projectID).Error; err == nil {
			envMap := make(map[string]string)
			
			// If appending, preserve existing env variables
			if mode == "append" && project.EnvironmentVariables != "" {
				json.Unmarshal([]byte(project.EnvironmentVariables), &envMap)
			}
			
			for _, v := range collection.Variable {
				envMap[v.Key] = v.Value
			}
			
			envBytes, _ := json.Marshal(envMap)
			project.EnvironmentVariables = string(envBytes)
			database.DB.Save(&project)
		}
	}

	return c.JSON(fiber.Map{
		"message": "Collection imported successfully",
		"count":   len(parsedAPIs),
	})
}

// PauseAPI allows a user to pause monitoring for a specific endpoint temporarily
func PauseAPI(c *fiber.Ctx) error {
	apiID := c.Params("id")
	userID := c.Locals("user_id").(uuid.UUID)
	role := c.Locals("role").(string)

	var api models.API
	if err := database.DB.First(&api, "id = ?", apiID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "API not found"})
	}

	var project models.Project
	if err := database.DB.Select("user_id").First(&project, "id = ?", api.ProjectID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Associated project not found"})
	}

	// Verify project ownership
	if role != "admin" && project.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Unauthorized to modify this API"})
	}

	type PauseInput struct {
		PauseHours float64 `json:"pause_hours"`
	}
	var input PauseInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	var baseAPI models.API
	database.DB.First(&baseAPI, "id = ?", apiID)
	
	if input.PauseHours > 0 {
		pausedTime := time.Now().Add(time.Duration(input.PauseHours * float64(time.Hour)))
		baseAPI.PausedUntil = &pausedTime
	} else if input.PauseHours < 0 {
		// Indefinite pause — set to a far future date
		indefinite := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
		baseAPI.PausedUntil = &indefinite
	} else {
		baseAPI.PausedUntil = nil
	}

	if err := database.DB.Save(&baseAPI).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update API pause status"})
	}

	return c.JSON(fiber.Map{
		"message":      "API pause status updated successfully",
		"paused_until": baseAPI.PausedUntil,
	})
}

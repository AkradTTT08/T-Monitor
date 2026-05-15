package utils

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/models"
)

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
		Raw   string `json:"raw"`
		Query []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"query"`
		Variable []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"variable"`
	} `json:"url"`
}

type PostmanItem struct {
	Name    string            `json:"name"`
	Request PostmanRequest    `json:"request"`
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

// ParsePostmanCollection parses a Postman Collection JSON file and returns a list of API models and environment variables
func ParsePostmanCollection(file io.Reader, projectID uuid.UUID) ([]models.API, map[string]string, error) {
	var collection PostmanCollection
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&collection); err != nil {
		return nil, nil, err
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
				method := item.Request.Method
				if method == "" {
					method = "GET"
				}

				// Handle Headers
				postmanHeaders := item.Request.Header

				// Case-insensitive check for Content-Type
				hasContentType := false
				for _, h := range postmanHeaders {
					if strings.EqualFold(h.Key, "Content-Type") {
						hasContentType = true
						break
					}
				}

				// Auto-add Content-Type if missing
				if !hasContentType {
					postmanHeaders = append(postmanHeaders, PostmanHeader{
						Key:   "Content-Type",
						Value: "application/json",
					})
				}

				headersJSON, _ := json.Marshal(postmanHeaders)

				// Handle Query Parameters and Path Variables
				type Param struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				}
				var allParams []Param

				// Add regular query params
				for _, q := range item.Request.URL.Query {
					if q.Key != "" {
						allParams = append(allParams, Param{Key: q.Key, Value: q.Value})
					}
				}
				// Add path variables
				for _, v := range item.Request.URL.Variable {
					if v.Key != "" {
						allParams = append(allParams, Param{Key: v.Key, Value: v.Value})
					}
				}

				params := "[]"
				if len(allParams) > 0 {
					pJSON, _ := json.Marshal(allParams)
					params = string(pJSON)
				}

				folderAssign := currentFolder
				if folderAssign == "" {
					folderAssign = "Uncategorized"
				}

				indefinite := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

				parsedAPIs = append(parsedAPIs, models.API{
					ProjectID:          projectID,
					Folder:             folderAssign,
					Name:               item.Name,
					Method:             method,
					URL:                item.Request.URL.Raw,
					Parameters:         params,
					Headers:            string(headersJSON),
					Body:               item.Request.Body.Raw,
					ExpectedStatusCode: 200,
					Interval:           60,
					PausedUntil:        &indefinite,
				})
			}
		}
	}

	parseItems(collection.Item, "")

	// Extract Environment Variables if defined in the collection
	envMap := make(map[string]string)
	if len(collection.Variable) > 0 {
		for _, v := range collection.Variable {
			envMap[v.Key] = v.Value
		}
	}

	return parsedAPIs, envMap, nil
}

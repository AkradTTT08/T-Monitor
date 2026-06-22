package workers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/monitor-api/backend/internal/models"
)

// ReplaceEnvVariables substitutes {{VAR}} with actual values from envVars
func ReplaceEnvVariables(input string, envVars map[string]string) string {
	if input == "" {
		return ""
	}
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	return re.ReplaceAllStringFunc(input, func(m string) string {
		key := m[2 : len(m)-2] // strip {{ and }}
		if val, ok := envVars[key]; ok {
			return val
		}
		return m
	})
}

// BuildRequest creates an http.Request from the API definition, replacing variables and building query params.
func BuildRequest(api models.API, envVars map[string]string) (*http.Request, error) {
	// 1. Variable Substitution
	targetURL := ReplaceEnvVariables(api.URL, envVars)
	headersStr := ReplaceEnvVariables(api.Headers, envVars)
	bodyStr := ReplaceEnvVariables(api.Body, envVars)
	paramsStr := ReplaceEnvVariables(api.Parameters, envVars)

	// 2. Parse Query Parameters and Path Variables
	if paramsStr != "" && paramsStr != "[]" && paramsStr != "{}" && paramsStr != "{\n}" {
		type Param struct {
			Key   string
			Value string
		}
		var allParams []Param

		// Try parsing as array of objects [{key, value}]
		var paramsArray []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal([]byte(paramsStr), &paramsArray); err == nil {
			for _, p := range paramsArray {
				if p.Key != "" {
					allParams = append(allParams, Param{Key: strings.TrimSpace(p.Key), Value: p.Value})
				}
			}
		} else {
			// Try parsing as map[string]interface{}
			var paramsMap map[string]interface{}
			if err := json.Unmarshal([]byte(paramsStr), &paramsMap); err == nil {
				for k, v := range paramsMap {
					if k != "" {
						allParams = append(allParams, Param{Key: strings.TrimSpace(k), Value: fmt.Sprintf("%v", v)})
					}
				}
			}
		}

		// Apply parameters: Path variables first ({id}, {{id}}, :id styles), then Query string
		if len(allParams) > 0 {
			queryParams := url.Values{}
			for _, p := range allParams {
				escaped := url.PathEscape(p.Value)
				// Support {{key}} style (double-brace, Postman-like)
				doubleBrace := "{{" + p.Key + "}}"
				// Support {key} style (single-brace, OpenAPI/Swagger-like)
				singleBrace := "{" + p.Key + "}"
				// Support :key style (Express/Fiber-like)
				colonStyle := ":" + p.Key

				if strings.Contains(targetURL, doubleBrace) {
					targetURL = strings.ReplaceAll(targetURL, doubleBrace, escaped)
				} else if strings.Contains(targetURL, singleBrace) {
					targetURL = strings.ReplaceAll(targetURL, singleBrace, escaped)
				} else if strings.Contains(targetURL, colonStyle) {
					targetURL = strings.ReplaceAll(targetURL, colonStyle, escaped)
				} else {
					// Not a path variable → append as query param
					queryParams.Add(p.Key, p.Value)
				}
			}
			
			// Append query params if they exist
			encodedQuery := queryParams.Encode()
			if encodedQuery != "" {
				if strings.Contains(targetURL, "?") {
					targetURL += "&" + encodedQuery
				} else {
					targetURL += "?" + encodedQuery
				}
			}
		}
	}

	// 3. Security Check
	if !IsSafeURL(targetURL) {
		return nil, fmt.Errorf("SSRF blocked: unsafe URL detected")
	}

	// 4. Create HTTP Request
	var reqBody []byte
	if bodyStr != "" {
		reqBody = []byte(stripJSONComments(bodyStr))
	}

	req, err := http.NewRequest(api.Method, targetURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 5. Add Headers
	if headersStr != "" && headersStr != "[]" && headersStr != "{}" {
		var headersArray []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal([]byte(headersStr), &headersArray); err == nil {
			for _, h := range headersArray {
				if h.Key != "" {
					req.Header.Set(strings.TrimSpace(h.Key), h.Value)
				}
			}
		} else {
			var headersMap map[string]string
			if err := json.Unmarshal([]byte(headersStr), &headersMap); err == nil {
				for k, v := range headersMap {
					if k != "" {
						req.Header.Set(strings.TrimSpace(k), v)
					}
				}
			}
		}
	}

	// Default Content-Type if not provided
	if req.Header.Get("Content-Type") == "" && reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// stripJSONComments removes // single-line and /* */ multi-line comments from a string,
// while correctly ignoring comment syntax inside quoted JSON string values.
func stripJSONComments(input string) string {
	runes := []rune(input)
	n := len(runes)
	var result []rune
	inString := false
	i := 0
	for i < n {
		ch := runes[i]
		// Track string boundaries, respecting backslash escapes
		if ch == '"' && (i == 0 || runes[i-1] != '\\') {
			inString = !inString
			result = append(result, ch)
			i++
			continue
		}
		if !inString {
			// Single-line comment: // ... \n
			if i+1 < n && ch == '/' && runes[i+1] == '/' {
				for i < n && runes[i] != '\n' {
					i++
				}
				continue
			}
			// Multi-line comment: /* ... */
			if i+1 < n && ch == '/' && runes[i+1] == '*' {
				i += 2
				for i+1 < n && !(runes[i] == '*' && runes[i+1] == '/') {
					i++
				}
				i += 2 // skip closing */
				continue
			}
		}
		result = append(result, ch)
		i++
	}
	return string(result)
}

package workers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dop251/goja"
	"github.com/monitor-api/backend/internal/models"
)

// EvaluateResult determines if the API call was successful based on status code and custom JS script.
func EvaluateResult(api models.API, resp *http.Response, bodyStr string, err error, duration time.Duration) (bool, string) {
	if err != nil {
		return false, fmt.Sprintf("Request failed: %v", err)
	}

	statusCode := resp.StatusCode
	expectedCode := api.ExpectedStatusCode
	if expectedCode == 0 {
		expectedCode = 200 // Default
	}

	// 1. Basic Status Code Check
	isSuccess := statusCode == expectedCode
	errorMessage := ""

	if !isSuccess {
		errorMessage = fmt.Sprintf("Expected status %d, got %d", expectedCode, statusCode)
	}

	// 2. Custom Response Script (JavaScript) Evaluation
	if api.ResponseScript != "" {
		scriptSuccess, scriptErr := runResponseScript(api.ResponseScript, statusCode, bodyStr, duration)
		if scriptErr != nil {
			// Script error: only mark failed if status was already OK
			// (don't double-penalise already-failed requests)
			if errorMessage != "" {
				errorMessage += "; "
			}
			errorMessage += fmt.Sprintf("Script Error: %v", scriptErr)
			// Script error is non-conclusive — keep status code decision
		} else {
			// Script returned a definitive boolean — use it
			isSuccess = scriptSuccess
			if !scriptSuccess {
				if errorMessage != "" {
					errorMessage += "; "
				}
				errorMessage += "Custom script validation failed"
			} else {
				// Script explicitly passed — clear any status code error
				errorMessage = ""
			}
		}
	}

	return isSuccess, errorMessage
}

// runResponseScript executes user-defined JavaScript to validate the response
func runResponseScript(script string, statusCode int, body string, responseTime time.Duration) (bool, error) {
	vm := goja.New()

	// Provide response data to the JS environment
	vm.Set("statusCode", statusCode)
	vm.Set("body", body)
	vm.Set("responseTime", responseTime.Milliseconds())

	// Run the script
	val, err := vm.RunString(script)
	if err != nil {
		return false, err
	}

	// The script should evaluate to a boolean
	if val != nil && val.Export() != nil {
		if result, ok := val.Export().(bool); ok {
			return result, nil
		}
		// Script returned a non-boolean value — treat as non-conclusive (no error)
		// Return true with no error so the status-code result is used
		return true, nil
	}

	// Script returned nothing (undefined) — treat as non-conclusive
	return true, nil
}

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
	success, msg, _ := EvaluateResultWithEnv(api, resp, bodyStr, err, duration)
	return success, msg
}

// EvaluateResultWithEnv is like EvaluateResult but also returns env variable updates
// captured from setenv() calls inside the response script.
func EvaluateResultWithEnv(api models.API, resp *http.Response, bodyStr string, err error, duration time.Duration) (bool, string, map[string]string) {
	if err != nil {
		return false, fmt.Sprintf("Request failed: %v", err), nil
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

	var envUpdates map[string]string

	// 2. Custom Response Script (JavaScript) Evaluation
	if api.ResponseScript != "" {
		scriptSuccess, scriptErr, updates := runResponseScriptWithEnv(api.ResponseScript, statusCode, bodyStr, duration)
		envUpdates = updates

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

	return isSuccess, errorMessage, envUpdates
}

// runResponseScript executes user-defined JavaScript to validate the response (legacy wrapper).
func runResponseScript(script string, statusCode int, body string, responseTime time.Duration) (bool, error) {
	success, err, _ := runResponseScriptWithEnv(script, statusCode, body, responseTime)
	return success, err
}

// runResponseScriptWithEnv executes the script and captures any setenv() calls.
func runResponseScriptWithEnv(script string, statusCode int, body string, responseTime time.Duration) (bool, error, map[string]string) {
	vm := goja.New()
	envUpdates := make(map[string]string)

	// Provide response data to the JS environment
	// Flat variables (original API)
	vm.Set("statusCode", statusCode)
	vm.Set("body", body)
	vm.Set("responseTime", responseTime.Milliseconds())

	// response object — allows scripts to use response.body, response.status, response.responseTime
	responseObj := vm.NewObject()
	responseObj.Set("body", body)
	responseObj.Set("status", statusCode)
	responseObj.Set("statusCode", statusCode)
	responseObj.Set("responseTime", responseTime.Milliseconds())
	vm.Set("response", responseObj)

	// setenv(key, value) — lets scripts persist values back to project ENV variables
	vm.Set("setenv", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) >= 2 {
			key := call.Arguments[0].String()
			value := call.Arguments[1].String()
			envUpdates[key] = value
		}
		return goja.Undefined()
	})

	// console.log / console.error — swallow gracefully so scripts don't crash
	console := vm.NewObject()
	console.Set("log", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	console.Set("error", func(call goja.FunctionCall) goja.Value { return goja.Undefined() })
	vm.Set("console", console)

	// Run the script
	val, err := vm.RunString(script)
	if err != nil {
		return false, err, envUpdates
	}

	// The script should evaluate to a boolean
	if val != nil && val.Export() != nil {
		if result, ok := val.Export().(bool); ok {
			return result, nil, envUpdates
		}
		// Script returned a non-boolean value — treat as non-conclusive (no error)
		// Return true with no error so the status-code result is used
		return true, nil, envUpdates
	}

	// Script returned nothing (undefined) — treat as non-conclusive
	return true, nil, envUpdates
}

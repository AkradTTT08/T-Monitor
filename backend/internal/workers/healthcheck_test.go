package workers_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/monitor-api/backend/internal/database"
	"github.com/monitor-api/backend/internal/models"
	"github.com/monitor-api/backend/internal/testutils"
	"github.com/monitor-api/backend/internal/workers"
	"github.com/stretchr/testify/assert"
)

// MockHTTPClient implements HTTPClient for testing
type MockHTTPClient struct {
	MockResponse *http.Response
	MockError    error
	LastRequest  *http.Request
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.LastRequest = req
	return m.MockResponse, m.MockError
}

func TestIsSafeURL(t *testing.T) {
	assert.False(t, workers.IsSafeURL("http://localhost:8080/api"))
	assert.False(t, workers.IsSafeURL("http://127.0.0.1/test"))
	assert.False(t, workers.IsSafeURL("http://10.0.0.5/internal"))
	assert.False(t, workers.IsSafeURL("http://192.168.1.1/router"))
	assert.False(t, workers.IsSafeURL("http://[::1]/ipv6"))
	
	assert.True(t, workers.IsSafeURL("https://www.google.com"))
	assert.True(t, workers.IsSafeURL("https://api.github.com/users"))
}

func TestEvaluator_SuccessAndFailure(t *testing.T) {
	api := models.API{ExpectedStatusCode: 200}
	
	// Test 1: Success
	respSuccess := &http.Response{StatusCode: 200}
	isSuccess, msg := workers.EvaluateResult(api, respSuccess, "{}", nil, time.Millisecond*50)
	assert.True(t, isSuccess)
	assert.Empty(t, msg)

	// Test 2: Failure (Wrong Status)
	respFail := &http.Response{StatusCode: 500}
	isSuccess, msg = workers.EvaluateResult(api, respFail, "Server Error", nil, time.Millisecond*50)
	assert.False(t, isSuccess)
	assert.Contains(t, msg, "Expected status 200, got 500")

	// Test 3: Failure (Network Error)
	isSuccess, msg = workers.EvaluateResult(api, nil, "", assert.AnError, time.Millisecond*50)
	assert.False(t, isSuccess)
	assert.Contains(t, msg, "Request failed")

	// Test 4: Custom Script overrides status failure
	apiWithScript := models.API{
		ExpectedStatusCode: 200,
		ResponseScript: `
			// If body contains "force_success", return true despite 500 status
			if (body.includes("force_success")) {
				true
			} else {
				false
			}
		`,
	}
	isSuccess, msg = workers.EvaluateResult(apiWithScript, respFail, `{"status": "force_success"}`, nil, time.Millisecond*50)
	assert.True(t, isSuccess) // Script passes
	assert.Empty(t, msg)
}

func TestRunPing_AutoRepairCreation(t *testing.T) {
	testutils.SetupTestDB()

	// 1. Setup Data
	user := models.User{Email: "worker@example.com"}
	database.DB.Create(&user)

	project := models.Project{Name: "Worker Project", UserID: user.ID}
	database.DB.Create(&project)

	api := models.API{
		Name: "Failing API", 
		ProjectID: project.ID, 
		Method: "GET", 
		URL: "http://example.com/fail",
		ExpectedStatusCode: 200,
	}
	database.DB.Create(&api)

	// 2. Mock HTTP Client to return 500
	mockClient := &MockHTTPClient{
		MockResponse: &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
		},
	}
	workers.SetTestClient(mockClient)

	// 3. Run Ping (This should fail and trigger an auto-repair task)
	workers.RunPing(api, nil, "Worker Project", "Company X")

	// 4. Verify MonitorLog was created
	var logEntry models.MonitorLog
	database.DB.Order("checked_at DESC").First(&logEntry, "api_id = ?", api.ID)
	assert.False(t, logEntry.IsSuccess)
	assert.Equal(t, 500, logEntry.StatusCode)
	assert.Contains(t, logEntry.ErrorMessage, "Expected status 200, got 500")

	// 5. Verify Auto-Repair Task was created
	var task models.RepairTask
	database.DB.First(&task, "api_id = ?", api.ID)
	assert.NotEqual(t, uuid.Nil, task.ID)
	assert.Equal(t, "open", task.Status)
	assert.Contains(t, task.Description, "Auto-detected failure for Failing API")

	// 6. Run Ping again, verify it doesn't create duplicate repair task
	workers.RunPing(api, nil, "Worker Project", "Company X")
	
	var taskCount int64
	database.DB.Model(&models.RepairTask{}).Where("api_id = ?", api.ID).Count(&taskCount)
	assert.Equal(t, int64(1), taskCount) // Still only 1 task
}

package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type ChatMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type AIService interface {
	AnalyzeIncident(errorMsg, responseBody, url string) (string, error)
	QueryData(query string, history []ChatMessage, dataContext string) (string, error)
}

type aiService struct {
	mockMode bool
}

func NewAIService(mockMode bool) AIService {
	return &aiService{mockMode: mockMode}
}

// getOllamaHost returns the Ollama base URL from env or default
func getOllamaHost() string {
	host := os.Getenv("OLLAMA_HOST")
	if host == "" {
		host = "http://localhost:11434"
	}
	return strings.TrimRight(host, "/")
}

// getOllamaModel returns the model name to use
func getOllamaModel() string {
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2"
	}
	return model
}

func (s *aiService) ollamaGenerate(prompt string) (string, error) {
	if s.mockMode {
		return "Mocked AI Response", nil
	}

	host := getOllamaHost()
	model := getOllamaModel()

	payload := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}

	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	log.Printf("[AI] Calling Ollama (%s) for prompt (len: %d)...", host, len(prompt))
	req, err := http.NewRequestWithContext(ctx, "POST", host+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Response string `json:"response"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("ollama error: %s", result.Error)
	}

	return result.Response, nil
}

func (s *aiService) AnalyzeIncident(errorMsg, responseBody, url string) (string, error) {
	prompt := fmt.Sprintf(`
You are a senior site reliability engineer (SRE) assisting a developer.
An API endpoint at %s has failed.

Error Message:
%s

Response Body:
%s

Please analyze this failure and provide a concise, actionable summary of what likely went wrong and what the developer should check first.
Do not wrap your response in markdown blocks or output anything else.
`, url, errorMsg, responseBody)

	return s.ollamaGenerate(prompt)
}

func (s *aiService) QueryData(query string, history []ChatMessage, dataContext string) (string, error) {
	var promptBuilder strings.Builder

	promptBuilder.WriteString("You are T-Monitor, an AI DevOps assistant. You help users analyze their API monitoring data. Here is the data context:\n")
	promptBuilder.WriteString(dataContext)
	promptBuilder.WriteString("\n\n--- Conversation History ---\n")

	for _, msg := range history {
		if msg.Role == "user" {
			promptBuilder.WriteString("User: " + msg.Text + "\n")
		} else {
			promptBuilder.WriteString("T-Monitor: " + msg.Text + "\n")
		}
	}

	promptBuilder.WriteString("User: " + query + "\n")
	promptBuilder.WriteString("T-Monitor: ")

	return s.ollamaGenerate(promptBuilder.String())
}

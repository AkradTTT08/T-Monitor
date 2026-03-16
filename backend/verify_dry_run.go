package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {
	baseURL := "http://localhost:8082/api/v1"
	email := "o.akrad.ttt08@gmail.com"
	password := "T@monitor123"

	// 1. Login to get token
	loginData := map[string]string{
		"email":    email,
		"password": password,
	}
	loginJSON, _ := json.Marshal(loginData)
	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewBuffer(loginJSON))
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var loginResult struct {
		Token string `json:"token"`
	}
	json.NewDecoder(resp.Body).Decode(&loginResult)
	token := loginResult.Token
	fmt.Println("Logged in successfully.")

	// 2. Test Dry Run via Query Parameter
	fmt.Println("\nTesting Dry Run via Query Parameter (?dry_run=true)...")
	projectData := map[string]interface{}{
		"name":        "DRY_RUN_TEST_PROJECT",
		"description": "This project should not be saved",
	}
	projectJSON, _ := json.Marshal(projectData)
	
	req, _ := http.NewRequest("POST", baseURL+"/projects?dry_run=true", bytes.NewBuffer(projectJSON))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))

	// 3. Test Dry Run via Header
	fmt.Println("\nTesting Dry Run via Header (X-Health-Check: T-Monitor)...")
	req, _ = http.NewRequest("POST", baseURL+"/projects", bytes.NewBuffer(projectJSON))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Health-Check", "T-Monitor")
	
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ = ioutil.ReadAll(resp.Body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))

	// 4. Verify that the project WAS NOT created
	fmt.Println("\nVerifying that project was NOT created in DB...")
	req, _ = http.NewRequest("GET", baseURL+"/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	var projects []interface{}
	json.NewDecoder(resp.Body).Decode(&projects)
	
	found := false
	for _, p := range projects {
		pm := p.(map[string]interface{})
		if pm["name"] == "DRY_RUN_TEST_PROJECT" {
			found = true
			break
		}
	}
	
	if found {
		fmt.Println("❌ ERROR: Project was found in the database!")
	} else {
		fmt.Println("✅ SUCCESS: Project was not found in the database.")
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
)

type ProjectInput struct {
	CompanyID *uuid.UUID `json:"company_id"`
}

func main() {
	jsonStr := `{"company_id":"b4d5a999-9999-9999-9999-fadd00000000"}`
	var input ProjectInput
	err := json.Unmarshal([]byte(jsonStr), &input)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if input.CompanyID == nil {
		fmt.Println("CompanyID is nil!")
	} else {
		fmt.Println("CompanyID:", input.CompanyID.String())
	}
}

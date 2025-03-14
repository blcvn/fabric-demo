package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const (
	apiURL      = "http://localhost:8080/account"
	startID     = 1001 // ID bắt đầu tạo
	numAccounts = 10   // Số lượng account muốn tạo
)

func main() {
	for i := 0; i < numAccounts; i++ {
		accountID := fmt.Sprintf("ACC%d", startID+i)
		if err := createAccount(accountID); err != nil {
			log.Printf("Failed to create account %s: %v", accountID, err)
		} else {
			log.Printf("Successfully created account: %s", accountID)
		}
	}
}

func createAccount(accountID string) error {
	url := fmt.Sprintf("%s/%s", apiURL, accountID)

	// Tạo request body
	requestBody, err := json.Marshal(map[string]string{"account_id": accountID})
	if err != nil {
		return err
	}

	// Gửi request POST đến API
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Kiểm tra phản hồi từ server
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

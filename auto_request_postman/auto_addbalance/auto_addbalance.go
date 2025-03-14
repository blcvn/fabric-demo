package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

const (
	apiURL      = "http://localhost:8080/balance/add"
	startID     = 1001 // ID bắt đầu
	numAccounts = 10   // Số lượng tài khoản cần cộng tiền
	amount      = 5000 // Số tiền cộng vào mỗi tài khoản
)

func main() {
	for i := 0; i < numAccounts; i++ {
		accountID := fmt.Sprintf("ACC%d", startID+i)
		if err := addBalance(accountID, amount); err != nil {
			log.Printf("Failed to add balance to account %s: %v", accountID, err)
		} else {
			log.Printf("Successfully added %d to account: %s", amount, accountID)
		}
	}
}

func addBalance(accountID string, amount int) error {
	// Tạo request body
	requestBody, err := json.Marshal(map[string]interface{}{
		"account_id": accountID,
		"amount":     amount,
	})
	if err != nil {
		return err
	}

	// Gửi request POST đến API
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(requestBody))
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

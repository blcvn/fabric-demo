package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

const baseURL = "http://localhost:8080/account/"

func createAccount(accountID int, wg *sync.WaitGroup) {
	defer wg.Done()

	url := fmt.Sprintf("%s%d", baseURL, accountID)
	requestBody, _ := json.Marshal(map[string]string{
		"account_id": fmt.Sprintf("user%d", accountID),
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("[ERROR] Failed to create account %d: %v\n", accountID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("[SUCCESS] Account %d created successfully\n", accountID)
	} else {
		fmt.Printf("[FAIL] Account %d failed with status: %d\n", accountID, resp.StatusCode)
	}
}

func main() {
	var wg sync.WaitGroup
	for i := 1001; i <= 1100; i++ {
		wg.Add(1)
		go createAccount(i, &wg)
	}
	wg.Wait()
	fmt.Println("All account creation requests completed.")
}

package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
)

const baseURL = "http://localhost:8080/account/"

var (
	mu       sync.Mutex
	accounts [][]string
)

func createAccount(accountID int, wg *sync.WaitGroup) {
	defer wg.Done()

	url := fmt.Sprintf("%s%d", baseURL, accountID)
	requestBody, _ := json.Marshal(map[string]string{
		"account_id": fmt.Sprintf("%d", accountID), // Không thêm "user"
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Printf("[ERROR] Failed to create account %d: %v\n", accountID, err)
		return
	}
	defer resp.Body.Close()

	// Lưu accountID vào danh sách mà không có status
	mu.Lock()
	accounts = append(accounts, []string{fmt.Sprintf("%d", accountID)})
	mu.Unlock()

	fmt.Printf("[INFO] Processed account %d\n", accountID)
}

func saveToCSV(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Println("Error creating CSV file:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Ghi tiêu đề cột
	writer.Write([]string{"account_id"})

	// Ghi dữ liệu
	writer.WriteAll(accounts)
	fmt.Println("CSV file saved:", filename)
}

func main() {
	var wg sync.WaitGroup
	for i := 1001; i <= 1100; i++ {
		wg.Add(1)
		go createAccount(i, &wg)
	}
	wg.Wait()

	// Lưu vào file CSV
	saveToCSV("accounts.csv")

	fmt.Println("All account creation requests completed.")
}

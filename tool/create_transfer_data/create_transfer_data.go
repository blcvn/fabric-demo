package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

const (
	inputFile    = "/home/lequocvieet/Desktop/longpk/fabric-demo/test_jmeter/csv_all_file/accounts.csv"
	outputFile   = "transfer_data.csv"
	numTransfers = 10000 // Số lượng giao dịch cần tạo
	minAmount    = 10    // Số tiền tối thiểu (số nguyên)
	maxAmount    = 1000  // Số tiền tối đa (số nguyên)
)

func main() {
	// Đọc danh sách tài khoản từ file account.csv
	accounts, err := readAccounts(inputFile)
	if err != nil {
		fmt.Println("Lỗi khi đọc file account.csv:", err)
		return
	}

	// Tạo file CSV output
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Lỗi khi tạo file transfer_data.csv:", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Ghi dòng tiêu đề
	writer.Write([]string{"from_account", "to_account", "amount"})

	// Sinh dữ liệu giao dịch ngẫu nhiên
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < numTransfers; i++ {
		fromIndex := rand.Intn(len(accounts))
		toIndex := rand.Intn(len(accounts))

		// Đảm bảo tài khoản nguồn và đích không trùng nhau
		for fromIndex == toIndex {
			toIndex = rand.Intn(len(accounts))
		}

		amount := rand.Intn(maxAmount-minAmount+1) + minAmount
		record := []string{accounts[fromIndex], accounts[toIndex], strconv.Itoa(amount)}
		writer.Write(record)
	}

	fmt.Println("File transfer_data.csv đã được tạo thành công!")
}

// Đọc danh sách tài khoản từ file CSV
func readAccounts(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var accounts []string
	for _, record := range records {
		if len(record) > 0 {
			accounts = append(accounts, record[0])
		}
	}

	return accounts, nil
}

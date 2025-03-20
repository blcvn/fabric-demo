package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
)

func main() {
	// Mở file CSV để ghi
	file, err := os.Create("balance_add.csv")
	if err != nil {
		fmt.Println("Error creating CSV file:", err)
		return
	}
	defer file.Close()

	// Tạo writer để ghi vào file CSV
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Ghi tiêu đề cột vào file CSV
	writer.Write([]string{"account_id", "balance_add"})

	// Ghi dữ liệu vào file CSV (account_id từ 1001 đến 1101 và DeductBalance là 1000)
	for i := 1; i <= 10000; i++ {
		accountID := strconv.Itoa(i)
		DeductBalance := "10000"
		writer.Write([]string{accountID, DeductBalance})
	}

	fmt.Println("CSV file 'balance_add.csv' has been generated successfully.")
}

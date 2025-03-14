package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const baseURL = "http://localhost:8080"

func main() {
	// Định dạng log có thời gian
	log.SetFlags(log.LstdFlags)

	// Đọc tham số dòng lệnh
	numAccounts := flag.Int("num-accounts", 10, "Số lượng tài khoản cần tạo")
	numTransactions := flag.Int("num-transactions", 50, "Số lượng giao dịch cần gửi")
	numWorkers := flag.Int("workers", 5, "Số lượng goroutines xử lý song song")
	flag.Parse()

	fmt.Printf("[%s] 🚀 Bắt đầu tạo %d tài khoản và gửi %d giao dịch...\n",
		getCurrentTime(), *numAccounts, *numTransactions)

	// 1. Tạo tài khoản (tránh trùng lặp)
	accounts := createAccounts(*numAccounts)

	// 2. Gửi giao dịch song song
	sendTransactions(accounts, *numTransactions, *numWorkers)

	fmt.Printf("[%s] ✅ Hoàn thành tất cả giao dịch!\n", getCurrentTime())
}

// Lấy thời gian hiện tại dưới dạng `YYYY-MM-DD HH:MM:SS`
func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// Gửi request HTTP POST
func httpPost(url string, payload interface{}) error {
	data, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s", resp.Status)
	}
	return nil
}

// Kiểm tra số dư tài khoản
func checkBalance(accountID string) (float64, error) {
	resp, err := http.Get(fmt.Sprintf("%s/balance/check?account_id=%s", baseURL, accountID))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Balance float64 `json:"balance"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return 0, err
	}
	return result.Balance, nil
}

// 1. Tạo tài khoản (tránh trùng lặp)
func createAccounts(num int) []string {
	var accounts []string
	accountSet := make(map[string]bool) // Lưu các account đã tạo
	startID := 1001                     // Bắt đầu từ 1001

	for len(accounts) < num {
		accountID := fmt.Sprintf("%d", startID) // Tạo số tài khoản

		// Kiểm tra nếu account đã tồn tại thì bỏ qua
		if accountSet[accountID] {
			startID++
			continue
		}

		// Gửi request tạo account
		err := httpPost(fmt.Sprintf("%s/account/%s", baseURL, accountID), nil)
		if err != nil {
			log.Printf("[%s] ❌ Lỗi tạo tài khoản %s: %v\n", getCurrentTime(), accountID, err)
			startID++
			continue
		}

		// Thêm vào danh sách và đánh dấu đã sử dụng
		accounts = append(accounts, accountID)
		accountSet[accountID] = true
		fmt.Printf("[%s] ✅ Tạo tài khoản: %s\n", getCurrentTime(), accountID)

		startID++ // Tăng số tài khoản tiếp theo
	}
	return accounts
}

// 2. Gửi giao dịch ngẫu nhiên
func sendTransactions(accounts []string, numTransactions int, numWorkers int) {
	var wg sync.WaitGroup
	jobs := make(chan int, numTransactions)

	// Khởi tạo worker
	for i := 0; i < numWorkers; i++ {
		go func() {
			for range jobs {
				randomTransaction(accounts)
				wg.Done()
			}
		}()
	}

	// Đưa công việc vào queue
	for i := 0; i < numTransactions; i++ {
		wg.Add(1)
		jobs <- i
	}

	// Chờ tất cả worker hoàn thành
	close(jobs)
	wg.Wait()
}

// Gửi giao dịch ngẫu nhiên
func randomTransaction(accounts []string) {
	rand.Seed(time.Now().UnixNano())

	// Chọn ngẫu nhiên loại giao dịch
	switch rand.Intn(3) {
	case 0:
		// AddBalance
		account := accounts[rand.Intn(len(accounts))]
		amount := rand.Float64() * 100
		err := httpPost(fmt.Sprintf("%s/balance/add", baseURL), map[string]interface{}{
			"account_id": account,
			"amount":     amount,
		})
		logTransaction("AddBalance", account, "", amount, err)
	case 1:
		// DeductBalance
		account := accounts[rand.Intn(len(accounts))]
		amount := rand.Float64() * 50
		balance, err := checkBalance(account)
		if err != nil {
			log.Printf("[%s] ❌ Lỗi kiểm tra số dư: %v\n", getCurrentTime(), err)
			return
		}
		if amount > balance {
			fmt.Printf("[%s] ❌ Giao dịch thất bại: Không đủ tiền trong tài khoản %s (%.2f)\n",
				getCurrentTime(), account, balance)
			return
		}
		err = httpPost(fmt.Sprintf("%s/balance/deduct", baseURL), map[string]interface{}{
			"account_id": account,
			"amount":     amount,
		})
		logTransaction("DeductBalance", account, "", amount, err)
	case 2:
		// TransferMoney
		from := accounts[rand.Intn(len(accounts))]
		to := accounts[rand.Intn(len(accounts))]
		for from == to { // Đảm bảo không chuyển cho chính mình
			to = accounts[rand.Intn(len(accounts))]
		}
		amount := rand.Float64() * 20

		// Kiểm tra số dư trước khi chuyển
		balance, err := checkBalance(from)
		if err != nil {
			log.Printf("[%s] ❌ Lỗi kiểm tra số dư: %v\n", getCurrentTime(), err)
			return
		}
		if amount > balance {
			fmt.Printf("[%s] ❌ Giao dịch thất bại: Không đủ tiền trong tài khoản %s (%.2f)\n",
				getCurrentTime(), from, balance)
			return
		}

		err = httpPost(fmt.Sprintf("%s/transfer", baseURL), map[string]interface{}{
			"from_account": from,
			"to_account":   to,
			"amount":       amount,
		})
		logTransaction("TransferMoney", from, to, amount, err)
	}
}

// Log giao dịch với timestamp
func logTransaction(action, from, to string, amount float64, err error) {
	timestamp := getCurrentTime()
	if err != nil {
		log.Printf("[%s] ❌ Lỗi %s: %v\n", timestamp, action, err)
	} else {
		if to == "" {
			fmt.Printf("[%s] ✅ %s thành công: %s -> %.2f\n", timestamp, action, from, amount)
		} else {
			fmt.Printf("[%s] ✅ %s thành công: %s → %s (%.2f)\n", timestamp, action, from, to, amount)
		}
	}
}

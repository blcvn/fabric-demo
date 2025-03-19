package chaincode

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SmartContract defines the chaincode
type SmartContract struct {
	contractapi.Contract
}

// Account defines the structure of an account
type Account struct {
	AccountID string  `json:"AccountID"`
	Balance   float64 `json:"balance"`
}

var accountLocks sync.Map

func getLock(accountID string) *sync.Mutex {
	lock, _ := accountLocks.LoadOrStore(accountID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// putStateWithRetry thực hiện ghi trạng thái với retry nếu gặp lỗi MVCC hoặc lỗi đồng thời.
func putStateWithRetry(ctx contractapi.TransactionContextInterface, key string, value []byte) error {
	maxRetries := 5
	var err error
	stub := ctx.GetStub()

	for i := 0; i < maxRetries; i++ {
		err = stub.PutState(key, value)
		if err == nil || (!isMVCCConflict(err) && !isConcurrencyError(err)) {
			return err
		}
		// Chờ một khoảng thời gian ngẫu nhiên để tránh xung đột liên tục
		time.Sleep(time.Duration(rand.Intn(100)+50) * time.Millisecond)
	}
	return err
}

func isMVCCConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "MVCC_READ_CONFLICT")
}

func isConcurrencyError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "concurrent transaction")
}

func (s *SmartContract) CreateAccount(ctx contractapi.TransactionContextInterface, accountID string) error {
	existingAccount, err := ctx.GetStub().GetState(accountID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if existingAccount != nil {
		return fmt.Errorf("account %s already exists", accountID)
	}

	account := Account{AccountID: accountID, Balance: 0}
	accountJSON, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account JSON: %v", err)
	}
	// Sử dụng putStateWithRetry để giảm khả năng lỗi MVCC
	return putStateWithRetry(ctx, accountID, accountJSON)
}

func (s *SmartContract) ReadAccount(ctx contractapi.TransactionContextInterface, accountID string) (*Account, error) {
	accountData, err := ctx.GetStub().GetState(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to read world state for %s: %v", accountID, err)
	}
	if accountData == nil {
		return nil, fmt.Errorf("account %s does not exist", accountID)
	}

	var account Account
	if err := json.Unmarshal(accountData, &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account %s: %v", accountID, err)
	}
	return &account, nil
}

func (s *SmartContract) AddBalance(ctx contractapi.TransactionContextInterface, accountID string, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}

	// 🔒 Lấy khóa tài khoản trước khi đọc dữ liệu
	lock := getLock(accountID)
	lock.Lock()
	defer lock.Unlock() // Luôn mở khóa khi xong

	// Đọc tài khoản từ world state
	account, err := s.ReadAccount(ctx, accountID)
	if err != nil && !strings.Contains(err.Error(), "does not exist") {
		return err
	}

	// Nếu tài khoản chưa tồn tại, tạo mới
	if account == nil {
		account = &Account{AccountID: accountID, Balance: 0}
	}

	// ✅ Cập nhật số dư
	account.Balance += amount

	// 🔄 Ghi trạng thái mới vào world state
	accountJSON, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal updated account JSON: %v", err)
	}
	return putStateWithRetry(ctx, accountID, accountJSON)
}

func (s *SmartContract) DeductBalance(ctx contractapi.TransactionContextInterface, accountID string, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be greater than zero")
	}

	lock := getLock(accountID)
	lock.Lock()
	defer lock.Unlock()

	account, err := s.ReadAccount(ctx, accountID)
	if err != nil {
		return err
	}
	if account.Balance < amount {
		return fmt.Errorf("insufficient balance")
	}

	account.Balance -= amount
	accountJSON, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal updated account JSON: %v", err)
	}
	return putStateWithRetry(ctx, accountID, accountJSON)
}

func (s *SmartContract) Transfer(ctx contractapi.TransactionContextInterface, fromID, toID string, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("transfer amount must be greater than zero")
	}
	if fromID == toID {
		return fmt.Errorf("cannot transfer to the same account")
	}

	// 🔒 Lấy khóa tài khoản nguồn trước
	fromLock := getLock(fromID)
	fromLock.Lock()
	defer fromLock.Unlock()

	// 📥 Đọc tài khoản nguồn
	fromAccount, err := s.ReadAccount(ctx, fromID)
	if err != nil {
		return err
	}
	if fromAccount.Balance < amount {
		return fmt.Errorf("insufficient balance in %s", fromID)
	}

	// 🔒 Lấy khóa tài khoản đích chỉ khi cần
	toLock := getLock(toID)
	toLock.Lock()
	defer toLock.Unlock()

	// 📤 Đọc tài khoản đích
	toAccount, err := s.ReadAccount(ctx, toID)
	if err != nil {
		toAccount = &Account{AccountID: toID, Balance: 0}
	}

	// 🔄 Cập nhật số dư
	fromAccount.Balance -= amount
	toAccount.Balance += amount

	// 💾 Ghi cập nhật vào world state
	fromJSON, _ := json.Marshal(fromAccount)
	toJSON, _ := json.Marshal(toAccount)

	if err := putStateWithRetry(ctx, fromID, fromJSON); err != nil {
		return fmt.Errorf("failed to update sender account: %v", err)
	}
	if err := putStateWithRetry(ctx, toID, toJSON); err != nil {
		return fmt.Errorf("failed to update recipient account: %v", err)
	}

	return nil
}

func (s *SmartContract) GetAllAccounts(ctx contractapi.TransactionContextInterface) ([]*Account, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var accounts []*Account
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var account Account
		if err := json.Unmarshal(queryResponse.Value, &account); err != nil {
			return nil, err
		}
		accounts = append(accounts, &account)
	}

	return accounts, nil
}

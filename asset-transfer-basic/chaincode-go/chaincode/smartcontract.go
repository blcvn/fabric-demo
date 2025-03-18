package chaincode

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
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
		return nil, fmt.Errorf("failed to read world state: %v", err)
	}
	if accountData == nil {
		return nil, fmt.Errorf("account %s does not exist", accountID)
	}

	var account Account
	if err := json.Unmarshal(accountData, &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %v", err)
	}
	return &account, nil
}

func (s *SmartContract) AddBalance(ctx contractapi.TransactionContextInterface, accountID string, amount float64) error {
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

	account.Balance += amount
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

func (s *SmartContract) TransferMoneyToMultiple(ctx contractapi.TransactionContextInterface, fromID string, transfers map[string]float64) error {
	if len(transfers) == 0 {
		return fmt.Errorf("no transfer targets provided")
	}

	// Lấy danh sách tài khoản cần khóa
	accountIDs := make([]string, 0, len(transfers)+1)
	accountIDs = append(accountIDs, fromID)
	for toID := range transfers {
		accountIDs = append(accountIDs, toID)
	}
	sort.Strings(accountIDs) // Sắp xếp để tránh deadlock

	// Khóa tất cả tài khoản theo thứ tự
	locks := make([]*sync.Mutex, len(accountIDs))
	for i, accID := range accountIDs {
		locks[i] = getLock(accID)
		locks[i].Lock()
	}
	defer func() {
		for _, lock := range locks {
			lock.Unlock()
		}
	}()

	// Đọc tài khoản nguồn
	fromAccount, err := s.ReadAccount(ctx, fromID)
	if err != nil {
		return err
	}

	totalAmount := 0.0
	for _, amount := range transfers {
		if amount <= 0 {
			return fmt.Errorf("transfer amount must be greater than zero")
		}
		totalAmount += amount
	}

	if fromAccount.Balance < totalAmount {
		return fmt.Errorf("insufficient balance in %s", fromID)
	}

	// Cập nhật tài khoản nguồn
	fromAccount.Balance -= totalAmount
	fromAccountJSON, err := json.Marshal(fromAccount)
	if err != nil {
		return fmt.Errorf("failed to marshal sender account JSON: %v", err)
	}
	if err := putStateWithRetry(ctx, fromID, fromAccountJSON); err != nil {
		return fmt.Errorf("failed to update sender account: %v", err)
	}

	// Cập nhật số dư các tài khoản nhận
	for toID, amount := range transfers {
		toAccount, err := s.ReadAccount(ctx, toID)
		if err != nil {
			// Nếu tài khoản chưa tồn tại, tạo mới
			toAccount = &Account{AccountID: toID, Balance: 0}
		}

		toAccount.Balance += amount
		toAccountJSON, err := json.Marshal(toAccount)
		if err != nil {
			return fmt.Errorf("failed to marshal recipient account JSON for %s: %v", toID, err)
		}
		if err := putStateWithRetry(ctx, toID, toAccountJSON); err != nil {
			return fmt.Errorf("failed to update recipient account %s: %v", toID, err)
		}
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

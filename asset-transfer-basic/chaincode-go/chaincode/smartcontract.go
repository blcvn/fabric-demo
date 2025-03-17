package chaincode

import (
	"encoding/json"
	"fmt"
	"sync"

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

// Map để quản lý các khóa tài khoản, đảm bảo an toàn khi giao dịch đồng thời
var accountLocks sync.Map

// getLock lấy mutex cho một tài khoản cụ thể
func getLock(accountID string) *sync.Mutex {
	lock, _ := accountLocks.LoadOrStore(accountID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

// CreateAccount tạo tài khoản mới với số dư ban đầu là 0
func (s *SmartContract) CreateAccount(ctx contractapi.TransactionContextInterface, accountID string) error {
	existingAccount, err := ctx.GetStub().GetState(accountID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if existingAccount != nil {
		return fmt.Errorf("account %s already exists", accountID)
	}

	account := Account{AccountID: accountID, Balance: 0}
	accountJSON, _ := json.Marshal(account)
	return ctx.GetStub().PutState(accountID, accountJSON)
}

// ReadAccount lấy thông tin tài khoản từ world state
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

// AddBalance thêm tiền vào tài khoản
func (s *SmartContract) AddBalance(ctx contractapi.TransactionContextInterface, accountID string, amount float64) error {
	lock := getLock(accountID)
	lock.Lock()
	defer lock.Unlock()

	account, err := s.ReadAccount(ctx, accountID)
	if err != nil {
		return err
	}

	account.Balance += amount
	accountJSON, _ := json.Marshal(account)
	return ctx.GetStub().PutState(accountID, accountJSON)
}

// DeductBalance trừ tiền khỏi tài khoản
func (s *SmartContract) DeductBalance(ctx contractapi.TransactionContextInterface, accountID string, amount float64) error {
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
	accountJSON, _ := json.Marshal(account)
	return ctx.GetStub().PutState(accountID, accountJSON)
}

// TransferMoneyToMultiple chuyển tiền từ một tài khoản nguồn đến nhiều tài khoản đích
func (s *SmartContract) TransferMoneyToMultiple(ctx contractapi.TransactionContextInterface, fromID string, transfers map[string]float64) error {
	fromLock := getLock(fromID)
	fromLock.Lock()
	defer fromLock.Unlock()

	fromAccount, err := s.ReadAccount(ctx, fromID)
	if err != nil {
		return err
	}

	// Tính tổng số tiền cần chuyển
	totalAmount := 0.0
	for _, amount := range transfers {
		totalAmount += amount
	}

	if fromAccount.Balance < totalAmount {
		return fmt.Errorf("insufficient balance in %s", fromID)
	}

	// Trừ tiền khỏi tài khoản nguồn trước khi thực hiện giao dịch
	fromAccount.Balance -= totalAmount
	stub := ctx.GetStub()
	if fromAccountJSON, err := json.Marshal(fromAccount); err == nil {
		if err := stub.PutState(fromID, fromAccountJSON); err != nil {
			return fmt.Errorf("failed to update fromAccount: %v", err)
		}
	} else {
		return fmt.Errorf("failed to marshal fromAccount: %v", err)
	}

	// Thực hiện cập nhật số dư tài khoản đích bằng Goroutine
	var wg sync.WaitGroup
	var mu sync.Mutex
	var transferErr error

	for toID, amount := range transfers {
		wg.Add(1)
		go func(toID string, amount float64) {
			defer wg.Done()
			toLock := getLock(toID)
			toLock.Lock()
			defer toLock.Unlock()

			account, err := s.ReadAccount(ctx, toID)
			if err != nil {
				mu.Lock()
				transferErr = fmt.Errorf("destination account %s does not exist", toID)
				mu.Unlock()
				return
			}

			account.Balance += amount

			if toAccountJSON, err := json.Marshal(account); err == nil {
				if err := stub.PutState(toID, toAccountJSON); err != nil {
					mu.Lock()
					transferErr = fmt.Errorf("failed to update toAccount %s: %v", toID, err)
					mu.Unlock()
				}
			} else {
				mu.Lock()
				transferErr = fmt.Errorf("failed to marshal toAccount %s: %v", toID, err)
				mu.Unlock()
			}
		}(toID, amount)
	}

	wg.Wait()

	if transferErr != nil {
		return transferErr
	}

	return nil
}

// GetAllAccounts lấy tất cả tài khoản trong world state
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

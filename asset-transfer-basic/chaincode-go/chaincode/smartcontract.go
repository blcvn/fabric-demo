package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// SmartContract defines the chaincode
type SmartContract struct {
	contractapi.Contract
}

// Account describes basic details of an account
type Account struct {
	AccountID string  `json:"AccountID"`
	Balance   float64 `json:"balance"`
}

func (s *SmartContract) CreateAccount(ctx contractapi.TransactionContextInterface, accountID string) error {
	accountData, err := ctx.GetStub().GetState(accountID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}
	if accountData != nil {
		return fmt.Errorf("account %s already exists", accountID)
	}

	account := Account{AccountID: accountID, Balance: 0}
	accountJSON, _ := json.Marshal(account)
	return ctx.GetStub().PutState(accountID, accountJSON)
}

// ReadAccount returns the account stored in world state
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

// AddBalance safely adds a specified amount to an account
func (s *SmartContract) AddBalance(ctx contractapi.TransactionContextInterface, accountID string, amount float64) error {
	account, err := s.ReadAccount(ctx, accountID)
	if err != nil {
		return err
	}

	account.Balance += amount
	accountJSON, _ := json.Marshal(account)
	return ctx.GetStub().PutState(accountID, accountJSON)
}

// DeductBalance safely deducts a specified amount from an account
func (s *SmartContract) DeductBalance(ctx contractapi.TransactionContextInterface, accountID string, amount float64) error {
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

// TransferMoney transfers funds between two accounts safely
func (s *SmartContract) TransferMoney(ctx contractapi.TransactionContextInterface, fromID, toID string, amount float64) error {
	fromAccount, err := s.ReadAccount(ctx, fromID)
	if err != nil {
		return err
	}
	if fromAccount.Balance < amount {
		return fmt.Errorf("insufficient balance in %s", fromID)
	}

	toAccount, err := s.ReadAccount(ctx, toID)
	if err != nil {
		return fmt.Errorf("destination account %s does not exist", toID)
	}

	// Cập nhật số dư tài khoản
	fromAccount.Balance -= amount
	toAccount.Balance += amount

	// Chuyển đổi dữ liệu JSON một lần
	fromAccountJSON, _ := json.Marshal(fromAccount)
	toAccountJSON, _ := json.Marshal(toAccount)

	// Cập nhật cả hai tài khoản trong một transaction (batch update)
	stub := ctx.GetStub()
	err = stub.PutState(fromID, fromAccountJSON)
	if err != nil {
		return fmt.Errorf("failed to update fromAccount: %v", err)
	}
	err = stub.PutState(toID, toAccountJSON)
	if err != nil {
		return fmt.Errorf("failed to update toAccount: %v", err)
	}

	return nil
}

// GetAllAccounts returns all accounts in the world state
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
		err = json.Unmarshal(queryResponse.Value, &account)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, &account)
	}

	return accounts, nil
}

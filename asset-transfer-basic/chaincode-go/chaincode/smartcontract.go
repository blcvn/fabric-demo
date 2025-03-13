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

// CreateAccount issues a new account with given accountID.
func (s *SmartContract) CreateAccount(ctx contractapi.TransactionContextInterface, accountID string) error {
	exists, err := s.AccountExists(ctx, accountID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the account %s already exists", accountID)
	}

	account := Account{
		AccountID: accountID,
		Balance:   0,
	}
	accountJSON, err := json.Marshal(account)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(accountID, accountJSON)
}

// ReadAccount returns the account stored in world state
func (s *SmartContract) ReadAccount(ctx contractapi.TransactionContextInterface, accountID string) (*Account, error) {
	accountJSON, err := ctx.GetStub().GetState(accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if accountJSON == nil {
		return nil, fmt.Errorf("the account %s does not exist", accountID)
	}

	var account Account
	err = json.Unmarshal(accountJSON, &account)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

// AccountExists checks if account exists
func (s *SmartContract) AccountExists(ctx contractapi.TransactionContextInterface, accountID string) (bool, error) {
	accountJSON, err := ctx.GetStub().GetState(accountID)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return accountJSON != nil, nil
}

// AddBalance safely adds a specified amount to an account
func (s *SmartContract) AddBalance(ctx contractapi.TransactionContextInterface, accountID string, amount float64) error {
	account, err := s.ReadAccount(ctx, accountID)
	if err != nil {
		return err
	}

	account.Balance += amount
	accountJSON, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %v", err)
	}

	return ctx.GetStub().PutState(accountID, accountJSON)
}

// TransferMoney transfers funds between two accounts safely
func (s *SmartContract) TransferMoney(ctx contractapi.TransactionContextInterface, fromAccountID, toAccountID string, amount float64) error {
	fromAccount, err := s.ReadAccount(ctx, fromAccountID)
	if err != nil {
		return err
	}
	if fromAccount.Balance < amount {
		return fmt.Errorf("insufficient balance in fromAccount")
	}

	toAccount, err := s.ReadAccount(ctx, toAccountID)
	if err != nil {
		return fmt.Errorf("destination account does not exist")
	}

	// Deduct and add balances safely
	fromAccount.Balance -= amount
	toAccount.Balance += amount

	fromAccountJSON, _ := json.Marshal(fromAccount)
	toAccountJSON, _ := json.Marshal(toAccount)

	// Atomic state update
	err = ctx.GetStub().PutState(fromAccountID, fromAccountJSON)
	if err != nil {
		return fmt.Errorf("failed to update fromAccount: %v", err)
	}

	err = ctx.GetStub().PutState(toAccountID, toAccountJSON)
	if err != nil {
		// Rollback fromAccount balance
		fromAccount.Balance += amount
		rollbackJSON, _ := json.Marshal(fromAccount)
		ctx.GetStub().PutState(fromAccountID, rollbackJSON)
		return fmt.Errorf("failed to update toAccount: %v", err)
	}

	return nil
}

// DeductBalance safely deducts a specified amount from an account
func (s *SmartContract) DeductBalance(ctx contractapi.TransactionContextInterface, accountID string, amount float64) error {
	account, err := s.ReadAccount(ctx, accountID)
	if err != nil {
		return err
	}
	if account.Balance < amount {
		return fmt.Errorf("insufficient balance in the account")
	}

	account.Balance -= amount
	accountJSON, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("failed to marshal account: %v", err)
	}

	return ctx.GetStub().PutState(accountID, accountJSON)
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

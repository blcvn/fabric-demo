package main

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

// Các hằng số cho kết nối và thông tin peer
const (
	mspID        = "Org1MSP"
	cryptoPath   = "../../test-network/organizations/peerOrganizations/org1.example.com"
	certPath     = cryptoPath + "/users/User1@org1.example.com/msp/signcerts"
	keyPath      = cryptoPath + "/users/User1@org1.example.com/msp/keystore"
	tlsCertPath  = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint = "dns:///localhost:7051"
	gatewayPeer  = "peer0.org1.example.com"
)

var now = time.Now()

func main() {
	// Tạo kết nối gRPC
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	// Tạo kết nối Gateway với Identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(clientConnection),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Cài đặt các tên channel và chaincode
	chaincodeName := "basic"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname
	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	// Các giao dịch với contract
	initLedger(contract)
	getAllAccounts(contract)
	createAccount(contract)
	readAccountByID(contract)
	transferMoney(contract)
	deductBalanceFromAccount(contract) // Gọi hàm trừ tiền từ tài khoản
	exampleErrorHandling(contract)
}

// Tạo kết nối gRPC
func newGrpcConnection() *grpc.ClientConn {
	certificatePEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		panic(fmt.Errorf("failed to read TLS certifcate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

// Tạo Identity
func newIdentity() *identity.X509Identity {
	certificatePEM, err := readFirstFile(certPath)
	if err != nil {
		panic(fmt.Errorf("failed to read certificate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

// Tạo Sign function
func newSign() identity.Sign {
	privateKeyPEM, err := readFirstFile(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}

// Đọc tệp đầu tiên trong thư mục
func readFirstFile(dirPath string) ([]byte, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}

	fileNames, err := dir.Readdirnames(1)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(path.Join(dirPath, fileNames[0]))
}

// Tạo ledger ban đầu
func initLedger(contract *client.Contract) {
	fmt.Printf("\n--> Submit Transaction: InitLedger, function creates the initial set of accounts on the ledger\n")

	_, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Lấy tất cả tài khoản
func getAllAccounts(contract *client.Contract) {
	fmt.Println("\n--> Evaluate Transaction: GetAllAccounts, function returns all the accounts in the ledger")

	evaluateResult, err := contract.EvaluateTransaction("GetAllAccounts")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

// Tạo tài khoản mới
func createAccount(contract *client.Contract) {
	assetId := fmt.Sprintf("account%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)
	fmt.Printf("\n--> Submit Transaction: CreateAccount, creates a new account\n")

	_, err := contract.SubmitTransaction("CreateAccount", assetId)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Đọc tài khoản theo ID
func readAccountByID(contract *client.Contract) {
	assetId := fmt.Sprintf("account%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)
	fmt.Printf("\n--> Evaluate Transaction: ReadAccount, function returns account details\n")

	evaluateResult, err := contract.EvaluateTransaction("ReadAccount", assetId)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

// Chuyển tiền giữa hai tài khoản
func transferMoney(contract *client.Contract) {
	fromAccountID := fmt.Sprintf("account%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)
	toAccountID := fmt.Sprintf("account%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)
	amount := 100.0

	fmt.Printf("\n--> Submit Transaction: TransferMoney, function transfers money between accounts\n")

	_, err := contract.SubmitTransaction("TransferMoney", fromAccountID, toAccountID, fmt.Sprintf("%f", amount))
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// DeductBalance safely deducts a specified amount from an account
func deductBalance(contract *client.Contract, accountID string, amount float64) {
	fmt.Printf("\n--> Submit Transaction: DeductBalance, function deducts money from account\n")

	// Submit transaction to deduct balance
	_, err := contract.SubmitTransaction("DeductBalance", accountID, fmt.Sprintf("%f", amount))
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Trừ tiền từ tài khoản
func deductBalanceFromAccount(contract *client.Contract) {
	accountID := fmt.Sprintf("account%d", now.Unix()*1e3+int64(now.Nanosecond())/1e6)
	amount := 50.0 // số tiền cần trừ

	// Gọi hàm DeductBalance
	deductBalance(contract, accountID, amount)
}

// Xử lý lỗi giao dịch
func exampleErrorHandling(contract *client.Contract) {
	fmt.Println("\n--> Submit Transaction: TransferMoney with insufficient balance")

	_, err := contract.SubmitTransaction("TransferMoney", "account70", "account71", "300")
	if err == nil {
		panic("******** FAILED to return an error")
	}

	fmt.Println("*** Successfully caught the error:")

	var endorseErr *client.EndorseError
	var submitErr *client.SubmitError
	var commitStatusErr *client.CommitStatusError
	var commitErr *client.CommitError

	if errors.As(err, &endorseErr) {
		fmt.Printf("Endorse error for transaction %s with gRPC status %v: %s\n", endorseErr.TransactionID, status.Code(endorseErr), endorseErr)
	} else if errors.As(err, &submitErr) {
		fmt.Printf("Submit error for transaction %s with gRPC status %v: %s\n", submitErr.TransactionID, status.Code(submitErr), submitErr)
	} else if errors.As(err, &commitStatusErr) {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("Timeout waiting for transaction %s commit status: %s", commitStatusErr.TransactionID, commitStatusErr)
		} else {
			fmt.Printf("Error obtaining commit status for transaction %s with gRPC status %v: %s\n", commitStatusErr.TransactionID, status.Code(commitStatusErr), commitStatusErr)
		}
	} else if errors.As(err, &commitErr) {
		fmt.Printf("Transaction %s failed to commit with status %d: %s\n", commitErr.TransactionID, int32(commitErr.Code), err)
	} else {
		panic(fmt.Errorf("unexpected error type %T: %w", err, err))
	}
}

// Định dạng JSON để dễ đọc
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}

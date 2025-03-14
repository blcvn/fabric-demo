package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	mspID         = "Org1MSP"
	cryptoPath    = "/home/lequocvieet/Desktop/longpk/fabric-demo/test-network/organizations/peerOrganizations/org1.example.com"
	certPath      = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/cert.pem"
	keyPath       = cryptoPath + "/users/User1@org1.example.com/msp/keystore"
	tlsCertPath   = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint  = "localhost:7051"
	channelName   = "mychannel"
	chaincodeName = "basic"
)

func main() {
	// Khởi tạo router Gin
	r := gin.Default()

	// Kết nối Fabric Gateway
	gateway, err := connectToFabricGateway()
	if err != nil {
		log.Fatalf("Failed to connect to Fabric Gateway: %v", err)
	}
	defer gateway.Close()

	network := gateway.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	// Định nghĩa các API
	r.POST("/account/:id", func(c *gin.Context) {
		accountID := c.Param("id")
		if err := createAccount(contract, accountID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Account created successfully"})
	})

	r.POST("/balance/add-batch", func(c *gin.Context) {
		var req struct {
			Accounts map[string]float64 `json:"accounts"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		type BalanceResult struct {
			BeforeBalance float64 `json:"beforeBalance"`
			Amount        float64 `json:"amount"`
			AfterBalance  float64 `json:"afterBalance"`
			Account       string  `json:"account"`
		}

		var wg sync.WaitGroup
		errChan := make(chan error, len(req.Accounts))
		resultChan := make(chan BalanceResult, len(req.Accounts))

		for accountID, amount := range req.Accounts {
			wg.Add(1)
			go func(accID string, amt float64) {
				defer wg.Done()
				beforeBalance, afterBalance, err := addBalance(contract, accID, amt)
				if err != nil {
					errChan <- fmt.Errorf("Account %s: %v", accID, err)
					return
				}
				resultChan <- BalanceResult{
					BeforeBalance: beforeBalance,
					Amount:        amt,
					AfterBalance:  afterBalance,
					Account:       accID,
				}
			}(accountID, amount)
		}

		wg.Wait()
		close(errChan)
		close(resultChan)

		var errors []string
		var results []BalanceResult

		for err := range errChan {
			errors = append(errors, err.Error())
		}
		for result := range resultChan {
			results = append(results, result)
		}

		if len(errors) > 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"errors": errors})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Batch transactions submitted successfully", "results": results})
	})

	r.GET("/account/:id", func(c *gin.Context) {
		accountID := c.Param("id")
		result, err := readAccount(contract, accountID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"account": string(result)})
	})

	r.POST("/balance/deduct", func(c *gin.Context) {
		var req struct {
			AccountID string  `json:"account_id"`
			Amount    float64 `json:"amount"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := deductBalance(contract, req.AccountID, req.Amount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Balance deducted successfully"})
	})

	r.POST("/transfer", func(c *gin.Context) {
		var req struct {
			FromAccount string  `json:"from_account"`
			ToAccount   string  `json:"to_account"`
			Amount      float64 `json:"amount"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := transferMoney(contract, req.FromAccount, req.ToAccount, req.Amount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Transfer successful"})
	})

	r.GET("/accounts", func(c *gin.Context) {
		result, err := getAllAccounts(contract)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var accounts []map[string]interface{}
		if err := json.Unmarshal(result, &accounts); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse JSON response"})
			return
		}

		c.IndentedJSON(http.StatusOK, gin.H{"accounts": accounts}) // Dùng IndentedJSON để format đẹp hơn
	})

	// Chạy server trên cổng 8080
	r.Run(":8080")
}

func connectToFabricGateway() (*client.Gateway, error) {
	clientConnection := newGrpcConnection()

	id, err := newIdentity()
	if err != nil {
		return nil, err
	}

	sign := newSigner()

	gateway, err := client.Connect(id, client.WithSign(sign), client.WithClientConnection(clientConnection))
	if err != nil {
		return nil, err
	}

	return gateway, nil
}

// === Các hàm gọi transaction ===

func createAccount(contract *client.Contract, accountID string) error {
	_, err := contract.SubmitTransaction("CreateAccount", accountID)
	if err != nil {
		if err.Error() == "rpc error: code = Aborted desc = failed to endorse transaction, see attached details for more info" {
			return fmt.Errorf("Account %s already exit", accountID)
		}
		return err
	}
	return nil
}

func addBalance(contract *client.Contract, accountID string, amount float64) (float64, float64, error) {
	// Đọc số dư trước khi thêm tiền
	beforeBalanceBytes, err := contract.EvaluateTransaction("ReadAccount", accountID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get before balance: %v", err)
	}

	var beforeData map[string]interface{}
	if err := json.Unmarshal(beforeBalanceBytes, &beforeData); err != nil {
		return 0, 0, fmt.Errorf("failed to parse before balance JSON: %v", err)
	}
	beforeBalance := beforeData["balance"].(float64)

	// Thực hiện giao dịch thêm tiền
	_, err = contract.SubmitTransaction("AddBalance", accountID, fmt.Sprintf("%.2f", amount))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to add balance: %v", err)
	}

	// Đọc lại số dư sau khi thêm tiền
	afterBalanceBytes, err := contract.EvaluateTransaction("ReadAccount", accountID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get after balance: %v", err)
	}

	var afterData map[string]interface{}
	if err := json.Unmarshal(afterBalanceBytes, &afterData); err != nil {
		return 0, 0, fmt.Errorf("failed to parse after balance JSON: %v", err)
	}
	afterBalance := afterData["balance"].(float64)

	return beforeBalance, afterBalance, nil
}

func readAccount(contract *client.Contract, accountID string) ([]byte, error) {
	return contract.EvaluateTransaction("ReadAccount", accountID)
}

func deductBalance(contract *client.Contract, accountID string, amount float64) error {
	_, err := contract.SubmitTransaction("DeductBalance", accountID, fmt.Sprintf("%.2f", amount))
	return err
}

func transferMoney(contract *client.Contract, fromAccountID, toAccountID string, amount float64) error {
	_, err := contract.SubmitTransaction("TransferMoney", fromAccountID, toAccountID, fmt.Sprintf("%.2f", amount))
	return err
}

func getAllAccounts(contract *client.Contract) ([]byte, error) {
	return contract.EvaluateTransaction("GetAllAccounts")
}

// === Kết nối gRPC ===
func newGrpcConnection() *grpc.ClientConn {
	certPool := x509.NewCertPool()
	caCert, _ := os.ReadFile(tlsCertPath)
	certPool.AppendCertsFromPEM(caCert)

	transportCredentials := credentials.NewClientTLSFromCert(certPool, "")
	connection, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		log.Fatalf("Failed to create gRPC connection: %v", err)
	}
	return connection
}

// === Xác thực danh tính ===
func newIdentity() (*identity.X509Identity, error) {
	certBytes, _ := os.ReadFile(certPath)
	block, _ := pem.Decode(certBytes)
	cert, _ := x509.ParseCertificate(block.Bytes)
	return identity.NewX509Identity(mspID, cert)
}

func newSigner() identity.Sign {
	keyFiles, _ := os.ReadDir(keyPath)
	keyBytes, _ := os.ReadFile(filepath.Join(keyPath, keyFiles[0].Name()))
	privateKey, _ := identity.PrivateKeyFromPEM(keyBytes)
	signer, _ := identity.NewPrivateKeySign(privateKey)
	return signer
}

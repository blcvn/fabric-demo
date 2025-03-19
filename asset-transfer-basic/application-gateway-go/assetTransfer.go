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
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	mspID         = "Org1MSP"
	cryptoPath    = "../../../fabric-demo/test-network/organizations/peerOrganizations/org1.example.com"
	certPath      = cryptoPath + "/users/User1@org1.example.com/msp/signcerts/cert.pem"
	keyPath       = cryptoPath + "/users/User1@org1.example.com/msp/keystore"
	tlsCertPath   = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint  = "localhost:7051"
	channelName   = "mychannel"
	chaincodeName = "basic" // Đổi tên chaincode nếu cần
)

func main() {
	// Khởi tạo router Gin
	r := gin.Default()

	// Kết nối tới Fabric Gateway
	gateway, err := connectToFabricGateway()
	if err != nil {
		log.Fatalf("Failed to connect to Fabric Gateway: %v", err)
	}
	defer gateway.Close()

	network := gateway.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	// API tạo tài khoản
	r.POST("/account/:id", func(c *gin.Context) {
		accountID := c.Param("id")
		_, err := contract.SubmitTransaction("CreateAccount", accountID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create account: %v", err)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Account created successfully"})
	})

	// API thêm tiền vào tài khoản
	r.POST("/balance/add", func(c *gin.Context) {
		var req struct {
			AccountID string  `json:"account_id"`
			Amount    float64 `json:"amount"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Lấy số dư hiện tại trước khi thêm tiền
		result, err := contract.EvaluateTransaction("ReadAccount", req.AccountID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read account: %v", err)})
			return
		}
		var account struct {
			Balance float64 `json:"balance"`
		}
		if err := json.Unmarshal(result, &account); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse account data"})
			return
		}
		beforeBalance := account.Balance

		// Chuyển đổi amount thành chuỗi (với định dạng có 2 số thập phân)
		amtStr := fmt.Sprintf("%.2f", req.Amount)
		_, err = contract.SubmitTransaction("AddBalance", req.AccountID, amtStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to add balance: %v", err)})
			return
		}

		// Tính toán số dư sau khi thêm tiền
		afterBalance := beforeBalance + req.Amount

		// Tạo cấu trúc kết quả
		resultData := struct {
			BeforeBalance float64 `json:"beforeBalance"`
			Amount        float64 `json:"amount"`
			AfterBalance  float64 `json:"afterBalance"`
			Account       string  `json:"account"`
		}{
			BeforeBalance: beforeBalance,
			Amount:        req.Amount,
			AfterBalance:  afterBalance,
			Account:       req.AccountID,
		}

		// Trả về phản hồi theo định dạng
		c.JSON(http.StatusOK, gin.H{
			"message": "Balance added successfully",
			"results": []interface{}{resultData},
		})
	})

	// API trừ tiền khỏi tài khoản
	r.POST("/balance/deduct", func(c *gin.Context) {
		var req struct {
			AccountID string  `json:"account_id" binding:"required"`
			Amount    float64 `json:"amount" binding:"required,gt=0"`
		}

		// Kiểm tra dữ liệu đầu vào
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Lấy số dư hiện tại trước khi trừ tiền
		result, err := contract.EvaluateTransaction("ReadAccount", req.AccountID)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Account %s not found", req.AccountID)})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read account: %v", err)})
			}
			return
		}

		var account struct {
			Balance float64 `json:"balance"`
		}
		if err := json.Unmarshal(result, &account); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse account data"})
			return
		}
		beforeDeduct := account.Balance

		// Kiểm tra nếu số dư hiện tại nhỏ hơn số tiền cần trừ
		if beforeDeduct < req.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient funds"})
			return
		}

		// Gọi Smart Contract để trừ tiền
		amtStr := fmt.Sprintf("%.2f", req.Amount)
		_, err = contract.SubmitTransaction("DeductBalance", req.AccountID, amtStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to deduct funds: %v", err)})
			return
		}

		// Tính toán số dư sau khi trừ tiền
		afterDeduct := beforeDeduct - req.Amount

		// Tạo cấu trúc kết quả
		resultData := struct {
			BeforeDeduct float64 `json:"beforeDeduct"`
			Amount       float64 `json:"amount"`
			AfterDeduct  float64 `json:"afterDeduct"`
			Account      string  `json:"account"`
		}{
			BeforeDeduct: beforeDeduct,
			Amount:       req.Amount,
			AfterDeduct:  afterDeduct,
			Account:      req.AccountID,
		}

		// Trả về phản hồi
		c.JSON(http.StatusOK, gin.H{
			"message": "Deduction successful",
			"results": []interface{}{resultData},
		})
	})

	// API chuyển tiền từ một tài khoản sang nhiều tài khoản
	r.POST("/transfer", func(c *gin.Context) {
		var req struct {
			FromAccount string  `json:"from_account" binding:"required"`
			ToAccount   string  `json:"to_account" binding:"required"`
			Amount      float64 `json:"amount" binding:"required,gt=0"`
		}

		// Kiểm tra dữ liệu đầu vào
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid request format. Ensure 'from_account', 'to_account' (strings) and 'amount' (positive number) are provided.",
				"details": err.Error(),
			})
			return
		}

		// Kiểm tra nếu người gửi và người nhận giống nhau
		if req.FromAccount == req.ToAccount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot transfer to the same account"})
			return
		}

		// Lấy số dư tài khoản nguồn trước khi chuyển khoản
		fromAccountData, err := contract.EvaluateTransaction("ReadAccount", req.FromAccount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read sender account: %v", err)})
			return
		}

		var fromAccount struct {
			Balance float64 `json:"balance"`
		}
		if err := json.Unmarshal(fromAccountData, &fromAccount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse sender account data"})
			return
		}

		// Kiểm tra số dư
		if fromAccount.Balance < req.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient funds in sender account"})
			return
		}

		// Thực hiện giao dịch chuyển tiền
		_, err = contract.SubmitTransaction("Transfer", req.FromAccount, req.ToAccount, fmt.Sprintf("%.2f", req.Amount))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to transfer funds: %v", err)})
			return
		}

		// Lấy số dư tài khoản sau giao dịch
		toAccountData, err := contract.EvaluateTransaction("ReadAccount", req.ToAccount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to read recipient account: %v", err)})
			return
		}

		var toAccount struct {
			Balance float64 `json:"balance"`
		}
		if err := json.Unmarshal(toAccountData, &toAccount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse recipient account data"})
			return
		}

		// Trả về phản hồi chi tiết
		resultData := struct {
			FromAccount   string  `json:"fromAccount"`
			ToAccount     string  `json:"toAccount"`
			Amount        float64 `json:"amount"`
			BeforeBalance float64 `json:"beforeBalance"`
			AfterBalance  float64 `json:"afterBalance"`
		}{
			FromAccount:   req.FromAccount,
			ToAccount:     req.ToAccount,
			Amount:        req.Amount,
			BeforeBalance: fromAccount.Balance,
			AfterBalance:  fromAccount.Balance - req.Amount,
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Transfer completed successfully",
			"results": resultData,
		})
	})

	// API đọc thông tin một tài khoản
	r.GET("/account/:id", func(c *gin.Context) {
		accountID := c.Param("id")
		result, err := contract.EvaluateTransaction("ReadAccount", accountID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to read account: %v", err)})
			return
		}
		var account map[string]interface{}
		if err := json.Unmarshal(result, &account); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse account data"})
			return
		}
		c.IndentedJSON(http.StatusOK, gin.H{"account": account})
	})

	// API lấy danh sách tất cả tài khoản
	r.GET("/accounts", func(c *gin.Context) {
		result, err := contract.EvaluateTransaction("GetAllAccounts")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to get accounts: %v", err)})
			return
		}
		var accounts []map[string]interface{}
		if err := json.Unmarshal(result, &accounts); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse accounts data"})
			return
		}
		c.IndentedJSON(http.StatusOK, gin.H{"accounts": accounts})
	})

	// Chạy server trên cổng 8080
	r.Run(":8080")
}

// Hàm kết nối tới Fabric Gateway thông qua gRPC
func connectToFabricGateway() (*client.Gateway, error) {
	clientConnection := newGrpcConnection()

	id, err := newIdentity()
	if err != nil {
		return nil, err
	}

	sign := newSigner()

	gw, err := client.Connect(id, client.WithSign(sign), client.WithClientConnection(clientConnection))
	if err != nil {
		return nil, err
	}

	return gw, nil
}

// Kết nối gRPC đến peer
func newGrpcConnection() *grpc.ClientConn {
	certPool := x509.NewCertPool()
	caCert, err := os.ReadFile(tlsCertPath)
	if err != nil {
		log.Fatalf("Failed to read TLS certificate: %v", err)
	}
	if !certPool.AppendCertsFromPEM(caCert) {
		log.Fatalf("Failed to append TLS certificate")
	}

	transportCredentials := credentials.NewClientTLSFromCert(certPool, "")
	connection, err := grpc.Dial(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		log.Fatalf("Failed to create gRPC connection: %v", err)
	}
	return connection
}

// Tạo danh tính (identity) từ file certificate
func newIdentity() (*identity.X509Identity, error) {
	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %v", err)
	}

	block, _ := pem.Decode(certBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from certificate")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse X.509 certificate: %v", err)
	}

	return identity.NewX509Identity(mspID, cert)
}

// Tạo signer từ private key
func newSigner() identity.Sign {
	keyFiles, err := os.ReadDir(keyPath)
	if err != nil || len(keyFiles) == 0 {
		log.Fatalf("Failed to read private key directory: %v", err)
	}

	keyBytes, err := os.ReadFile(filepath.Join(keyPath, keyFiles[0].Name()))
	if err != nil {
		log.Fatalf("Failed to read private key file: %v", err)
	}

	privateKey, err := identity.PrivateKeyFromPEM(keyBytes)
	if err != nil {
		log.Fatalf("Failed to parse private key: %v", err)
	}

	signer, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		log.Fatalf("Failed to create signer: %v", err)
	}

	return signer
}

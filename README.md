# Test-Network Hyperledger Fabric

## 1. Cài Đặt Môi Trường
Trước tiên, bạn cần cài đặt Hyperledger Fabric và các công cụ đi kèm.

### 📌 Cài đặt các yêu cầu cần thiết:
- **Docker** và **Docker Compose**
- **Go** (nếu dùng Smart Contract bằng Go)
- **Node.js** (nếu dùng Smart Contract bằng JavaScript/TypeScript)
- **Java** (nếu dùng Smart Contract bằng Java)

---

## 2. Tải Fabric Samples và Cài Đặt
```sh
git clone https://github.com/blcvn/fabric-demo.git
cd fabric-samples  
curl -sSL https://bit.ly/2ysbOFE | bash -s
ls -l Hyperledger-Fabric/bin
export PATH=$PATH:$(pwd)/bin
export FABRIC_CFG_PATH=$(pwd)/config
echo $PATH
echo $FABRIC_CFG_PATH
```

---

## 3. Khởi Chạy Mạng Thử Nghiệm
```sh
cd ~/Hyperledger-Fabric/test-network
./network.sh down # Dừng network nếu đang chạy
./network.sh up createChannel -ca
peer lifecycle chaincode package basic.tar.gz --path ../asset-transfer-basic/chaincode-go --lang golang --label basic_1.0
./network.sh deployCC -ccn basic -ccp ../asset-transfer-basic/chaincode-go -ccl go
```
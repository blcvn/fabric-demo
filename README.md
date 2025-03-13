Test-Network Hyperledger Fabric
1. Cài Đặt Môi Trường
Trước tiên, bạn cần cài đặt Hyperledger Fabric và các công cụ đi kèm.
📌 Cài đặt các yêu cầu cần thiết:
Docker và Docker Compose
Go (nếu dùng Smart Contract bằng Go)
Node.js (nếu dùng Smart Contract bằng JavaScript/TypeScript)
Java (nếu dùng Smart Contract bằng Java)
2. Tải Fabric Samples và Cài Đặt
git clone https://github.com/hyperledger/fabric-samples.git 
cd fabric-samples  
curl -sSL https://bit.ly/2ysbOFE | bash -s
ls -l Hyperledger-Fabric/bin
export PATH=$PATH:$(pwd)/bin
export FABRIC_CFG_PATH=$(pwd)/config
echo $PATH
echo $FABRIC_CFG_PATH
3. Khởi Chạy Mạng Thử Nghiệm
cd ~/Hyperledger-Fabric/test-network
./network.sh down # Dừng network nếu đang chạy
./network.sh up createChannel -ca
4. Package Chaincode
cd /Users/phamkhanhlong/Hyperledger-Fabric/test-network
peer lifecycle chaincode package basic.tar.gz --path ../asset-transfer-basic/chaincode-go --lang golang --label basic_1.0
5. Cài Đặt Chaincode Mới Lên Peer
cd ~/Hyperledger-Fabric/test-network
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
peer lifecycle chaincode install basic.tar.gz

peer lifecycle chaincode queryinstalled

=> peer lifecycle chaincode install basic.tar.gz
export CORE_PEER_ADDRESS=localhost:9051
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
peer lifecycle chaincode install basic.tar.gz
peer lifecycle chaincode queryinstalled
=> peer lifecycle chaincode install basic.tar.gz
peer lifecycle chaincode queryinstalled
6. Phê duyệt Chaincode
	cd ~/Hyperledger-Fabric/test-network
Phê duyệt chaincode cho Org1MSP
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export CORE_PEER_MSPCONFIGPATH=$PWD/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051

peer lifecycle chaincode approveformyorg -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com --tls \
--cafile $PWD/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem \
--channelID mychannel --name basic --version 1.0 \
--package-id basic_1.0:383b86cb83eb48ed18776cf87c219064dd499b0295695b8fb08b0929f758b410 --sequence 1

Phê duyệt chaincode cho Org2MSP
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem
export CORE_PEER_MSPCONFIGPATH=$PWD/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051

peer lifecycle chaincode approveformyorg -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com --tls \
--cafile $PWD/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem \
--channelID mychannel --name basic --version 1.0 \
--package-id basic_1.0:383b86cb83eb48ed18776cf87c219064dd499b0295695b8fb08b0929f758b410 --sequence 1
7. Commit chaincode:
Sau khi cả hai tổ chức đã phê duyệt, chạy lệnh commit:
peer lifecycle chaincode commit -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com --tls \
--cafile $PWD/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem \
--channelID mychannel --name basic --version 1.0 --sequence 1 \
--peerAddresses localhost:7051 \
--tlsRootCertFiles $PWD/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem \
--peerAddresses localhost:9051 \
--tlsRootCertFiles $PWD/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem
Lệnh kiểm tra: peer lifecycle chaincode querycommitted -C mychannel -n basic



8. Gọi hàm để khởi tạo dữ liệu
Tạo tài khoản mới (CreateAccount)
peer chaincode invoke -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com --tls \
--cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
-C mychannel -n basic \
--peerAddresses localhost:7051 \
--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
--peerAddresses localhost:9051 \
--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
-c '{"function":"CreateAccount","Args":["1001"]}' \
--waitForEvent
Kiểm tra thông tin tài khoản (ReadAccount)
peer chaincode query -C mychannel -n basic -c '{"function":"ReadAccount","Args":["1001"]}'
Kiểm tra tài khoản có tồn tại không (AccountExists)
	peer chaincode query -C mychannel -n basic -c '{"function":"AccountExists","Args":["1002"]}'
Nạp tiền vào tài khoản (AddBalance)
peer chaincode invoke -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com --tls \
--cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
-C mychannel -n basic \
--peerAddresses localhost:7051 \
--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
--peerAddresses localhost:9051 \
--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
-c '{"function":"AddBalance","Args":["1001", "10000"]}' \
--waitForEvent
Chuyển tiền từ a sang b:
peer chaincode invoke -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com --tls \
--cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
-C mychannel -n basic \
--peerAddresses localhost:7051 \
--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
--peerAddresses localhost:9051 \
--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
-c '{"function":"TransferMoney","Args":["1001", "1002", "5000"]}' \
--waitForEvent
Kiểm tra số dư sau khi chuyển tiền (ReadAccount)
peer chaincode query -C mychannel -n basic -c '{"function":"ReadAccount","Args":["1001"]}'
peer chaincode query -C mychannel -n basic -c '{"function":"ReadAccount","Args":["acc456"]}'
Kiểm tra toàn bộ dữ liệu
peer chaincode query -C mychannel -n basic -c '{"function":"GetAllAccounts","Args":[]}'
Gọi hàm DeductBalance trong CLI
peer chaincode invoke -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com --tls \
--cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem \
-C mychannel -n basic \
--peerAddresses localhost:7051 \
--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt \
--peerAddresses localhost:9051 \
--tlsRootCertFiles ${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt \
-c '{"function":"DeductBalance","Args":["1001", "1000"]}' \
--waitForEvent

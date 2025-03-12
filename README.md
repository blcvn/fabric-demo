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
ls -l fabric-samples/bin
export PATH=$PATH:$(pwd)/bin
export FABRIC_CFG_PATH=$(pwd)/config
echo $PATH
echo $FABRIC_CFG_PATH
3. Khởi Chạy Mạng Thử Nghiệm
cd ~/fabric-samples/test-network
./network.sh down # Dừng network nếu đang chạy
./network.sh up createChannel -ca
4. Package Chaincode
cd /Users/phamkhanhlong/fabric-samples/test-network
peer lifecycle chaincode package basic.tar.gz --path ../asset-transfer-basic/chaincode-go --lang golang --label basic_1.0
export CORE_PEER_MSPCONFIGPATH=$HOME/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_TLS_ROOTCERT_FILE=$HOME/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
5. Cài Đặt Chaincode Mới Lên Peer
cd ~/fabric-samples/test-network
peer lifecycle chaincode package basic.tar.gz --path ../chaincode-go --lang golang --label basic_1.0
peer lifecycle chaincode queryinstalled
6. Phê duyệt Chaincode:
- Phê duyệt chaincode cho Org1MSP:
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem
export CORE_PEER_MSPCONFIGPATH=$PWD/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
-----------
peer lifecycle chaincode approveformyorg -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com --tls \
--cafile $PWD/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem \
--channelID mychannel --name basic --version 1.0 \
--package-id basic_1.0:b2974d4b301358c998b17154ea19f8376cd812fddfd96150a17888520322630d --sequence 1
------------
Phê duyệt chaincode cho Org2MSP:
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=$PWD/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem
export CORE_PEER_MSPCONFIGPATH=$PWD/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051
------------
peer lifecycle chaincode approveformyorg -o localhost:7050 \
--ordererTLSHostnameOverride orderer.example.com --tls \
--cafile $PWD/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem \
--channelID mychannel --name basic --version 1.0 \
--package-id basic_1.0: b2974d4b301358c998b17154ea19f8376cd812fddfd96150a17888520322630d --sequence 1
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
8. Gọi hàm InitLedger để khởi tạo dữ liệu:

peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls \
--cafile /Users/phamkhanhlong/fabric-samples/test-network/organizations/ordererOrganizations/example.com/tlsca/tlsca.example.com-cert.pem \
-C mychannel -n basic --peerAddresses localhost:7051 \
--tlsRootCertFiles /Users/phamkhanhlong/fabric-samples/test-network/organizations/peerOrganizations/org1.example.com/tlsca/tlsca.org1.example.com-cert.pem \
--peerAddresses localhost:9051 \
--tlsRootCertFiles /Users/phamkhanhlong/fabric-samples/test-network/organizations/peerOrganizations/org2.example.com/tlsca/tlsca.org2.example.com-cert.pem \
-c '{"Args":["InitLedger"]}'
peer chaincode query -C mychannel -n basic -c '{"Args":["GetAllAssets"]}'


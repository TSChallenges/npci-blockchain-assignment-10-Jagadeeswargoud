#!/bin/bash

# Register drug
peer chaincode invoke -o orderer.example.com:7050 --tls --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem -C mychannel -n drugChaincode -c '{"function":"RegisterDrug","Args":["AZD1001","Paracetamol","BATCH001","2023-01-01","2025-01-01","Paracetamol 500mg"]}' --peerAddresses peer0.cipla.example.com:7051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/cipla.example.com/peers/peer0.cipla.example.com/tls/ca.crt

# Ship drug
peer chaincode invoke -o orderer.example.com:7050 --tls --cafile ${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem -C mychannel -n drugChaincode -c '{"function":"ShipDrug","Args":["AZD1001","Medlife"]}' --peerAddresses peer0.cipla.example.com:7051 --tlsRootCertFiles ${PWD}/organizations/peerOrganizations/cipla.example.com/peers/peer0.cipla.example.com/tls/ca.crt

# ... add other test transactions
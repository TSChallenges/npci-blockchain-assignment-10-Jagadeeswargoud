#!/bin/bash

# Track drug history
peer chaincode query -C mychannel -n drugChaincode -c '{"function":"TrackDrug","Args":["AZD1001"]}'

# Verify authenticity
peer chaincode query -C mychannel -n drugChaincode -c '{"function":"VerifyAuthenticity","Args":["AZD1001"]}'
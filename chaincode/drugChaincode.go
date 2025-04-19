package main

import (
	"encoding/json"
	"fmt"
	"time"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type Drug struct {
	DrugID          string   `json:"drugId"`
	Name            string   `json:"name"`
	Manufacturer    string   `json:"manufacturer"`
	BatchNumber     string   `json:"batchNumber"`
	MfgDate         string   `json:"mfgDate"`
	ExpiryDate      string   `json:"expiryDate"`
	Composition     string   `json:"composition"`
	CurrentOwner    string   `json:"currentOwner"` // Cipla, Medlife, Apollo
	Status          string   `json:"status"`       // InProduction, InTransit, Delivered, Recalled
	History         []string `json:"history"`      // Format: "timestamp|event|from|to|details"
	IsRecalled      bool     `json:"isRecalled"`
	InspectionNotes []string `json:"inspectionNotes"`
}

type SmartContract struct {
	contractapi.Contract
}

// ============== HELPER FUNCTIONS ==============
func getClientMSPID(ctx contractapi.TransactionContextInterface) (string, error) {
	id, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return "", fmt.Errorf("failed to get client MSPID: %v", err)
	}
	return id, nil
}

func getTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

func (s *SmartContract) drugExists(ctx contractapi.TransactionContextInterface, drugID string) (bool, error) {
	drugJSON, err := ctx.GetStub().GetState(drugID)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return drugJSON != nil, nil
}

// ============== MANUFACTURER FUNCTIONS ==============
func (s *SmartContract) RegisterDrug(ctx contractapi.TransactionContextInterface, 
	drugID string, name string, batchNumber string, mfgDate string, expiryDate string, composition string) error {
	
	// Verify caller is CiplaMSP
	clientMSPID, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}
	if clientMSPID != "CiplaMSP" {
		return fmt.Errorf("only Cipla can register drugs")
	}

	// Check if drug exists
	exists, err := s.drugExists(ctx, drugID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("drug %s already exists", drugID)
	}

	// Initialize drug object
	drug := Drug{
		DrugID:       drugID,
		Name:         name,
		Manufacturer: "Cipla",
		BatchNumber:  batchNumber,
		MfgDate:      mfgDate,
		ExpiryDate:   expiryDate,
		Composition:  composition,
		CurrentOwner: "Cipla",
		Status:       "InProduction",
		History: []string{
			fmt.Sprintf("%s|Registered|SYSTEM|Cipla|Drug registered in system", getTimestamp()),
		},
		IsRecalled:      false,
		InspectionNotes: []string{},
	}

	// Save to ledger
	drugJSON, err := json.Marshal(drug)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(drugID, drugJSON)
}

// ============== DISTRIBUTION FUNCTIONS ==============
func (s *SmartContract) ShipDrug(ctx contractapi.TransactionContextInterface, drugID string, to string) error {
	// Verify current owner is caller
	clientMSPID, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	// Get drug from ledger
	drugJSON, err := ctx.GetStub().GetState(drugID)
	if err != nil {
		return fmt.Errorf("failed to read drug: %v", err)
	}
	if drugJSON == nil {
		return fmt.Errorf("drug %s does not exist", drugID)
	}

	var drug Drug
	err = json.Unmarshal(drugJSON, &drug)
	if err != nil {
		return err
	}

	// Verify ownership
	if drug.CurrentOwner != clientMSPID {
		return fmt.Errorf("only current owner can ship drug")
	}

	// Update drug
	drug.CurrentOwner = to
	drug.Status = "InTransit"
	historyEntry := fmt.Sprintf("%s|Shipped|%s|%s|Drug shipment initiated", 
		getTimestamp(), drug.CurrentOwner, to)
	drug.History = append(drug.History, historyEntry)

	// Save updated drug
	updatedDrugJSON, err := json.Marshal(drug)
	if err != nil {
		return err
	}

	// Emit event
	eventPayload := fmt.Sprintf("Drug %s shipped from %s to %s", drugID, clientMSPID, to)
	ctx.GetStub().SetEvent("Shipment", []byte(eventPayload))

	return ctx.GetStub().PutState(drugID, updatedDrugJSON)
}

// ============== RECEIVE DRUG FUNCTION ==============
func (s *SmartContract) ReceiveDrug(ctx contractapi.TransactionContextInterface, drugID string) error {
	clientMSPID, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}

	// Get drug from ledger
	drugJSON, err := ctx.GetStub().GetState(drugID)
	if err != nil {
		return fmt.Errorf("failed to read drug: %v", err)
	}
	if drugJSON == nil {
		return fmt.Errorf("drug %s does not exist", drugID)
	}

	var drug Drug
	err = json.Unmarshal(drugJSON, &drug)
	if err != nil {
		return err
	}

	// Verify drug is in transit to this organization
	if drug.Status != "InTransit" || drug.CurrentOwner != clientMSPID {
		return fmt.Errorf("drug is not in transit to this organization")
	}

	// Update drug status
	drug.Status = "Delivered"
	historyEntry := fmt.Sprintf("%s|Received|%s|%s|Drug received", 
		getTimestamp(), drug.CurrentOwner, clientMSPID)
	drug.History = append(drug.History, historyEntry)

	// Save updated drug
	updatedDrugJSON, err := json.Marshal(drug)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(drugID, updatedDrugJSON)
}

// ============== REGULATOR FUNCTIONS ==============
func (s *SmartContract) RecallDrug(ctx contractapi.TransactionContextInterface, drugID string, reason string) error {
	// Verify caller is CDSCOMSP
	clientMSPID, err := getClientMSPID(ctx)
	if err != nil {
		return err
	}
	if clientMSPID != "CDSCOMSP" {
		return fmt.Errorf("only regulators can recall drugs")
	}

	// Get drug from ledger
	drugJSON, err := ctx.GetStub().GetState(drugID)
	if err != nil {
		return fmt.Errorf("failed to read drug: %v", err)
	}
	if drugJSON == nil {
		return fmt.Errorf("drug %s does not exist", drugID)
	}

	var drug Drug
	err = json.Unmarshal(drugJSON, &drug)
	if err != nil {
		return err
	}

	// Update drug status
	drug.IsRecalled = true
	drug.Status = "Recalled"
	drug.InspectionNotes = append(drug.InspectionNotes, 
		fmt.Sprintf("Recall on %s: %s", getTimestamp(), reason))
	historyEntry := fmt.Sprintf("%s|Recalled|%s|ALL|%s", 
		getTimestamp(), clientMSPID, reason)
	drug.History = append(drug.History, historyEntry)

	// Save updated drug
	updatedDrugJSON, err := json.Marshal(drug)
	if err != nil {
		return err
	}

	// Emit recall event
	ctx.GetStub().SetEvent("Recall", []byte(fmt.Sprintf("Drug %s recalled: %s", drugID, reason)))

	return ctx.GetStub().PutState(drugID, updatedDrugJSON)
}

// ============== COMMON FUNCTIONS ==============
func (s *SmartContract) TrackDrug(ctx contractapi.TransactionContextInterface, drugID string) (string, error) {
	drugJSON, err := ctx.GetStub().GetState(drugID)
	if err != nil {
		return "", fmt.Errorf("failed to read drug: %v", err)
	}
	if drugJSON == nil {
		return "", fmt.Errorf("drug %s does not exist", drugID)
	}
	return string(drugJSON), nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		fmt.Printf("Error creating chaincode: %s", err.Error())
		return
	}
	if err := chaincode.Start(); err != nil {
		fmt.Printf("Error starting chaincode: %s", err.Error())
	}
}
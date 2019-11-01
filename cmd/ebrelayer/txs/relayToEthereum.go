package txs

import (
	"context"
	"fmt"

	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/cosmos/peggy/cmd/ebrelayer/events"
	cosmosBridge "github.com/cosmos/peggy/cmd/ebrelayer/generated/cosmosbridge"
	oracle "github.com/cosmos/peggy/cmd/ebrelayer/generated/oracle"
)

const (
	// GasLimit : the gas limit in Gwei used for transactions sent with TransactOpts
	GasLimit = uint64(600000)
)

// RelayProphecyClaimToEthereum : relays the provided ProphecyClaim to the Ethereum network
func RelayProphecyClaimToEthereum(provider string, contractAddress common.Address, event events.Event, claim ProphecyClaim) error {

	// Start Ethereum client
	client, err := ethclient.Dial(provider)
	if err != nil {
		log.Fatal(err)
	}

	// Load the validator's private key
	key, err := LoadPrivateKey()
	if err != nil {
		log.Fatal(err)
	}

	// Load the validator's address
	sender, err := LoadSender()
	if err != nil {
		log.Fatal(err)
	}

	nonce, err := client.PendingNonceAt(context.Background(), sender)
	if err != nil {
		log.Fatal(err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Set up TransactOpts auth's tx signature authorization
	transactOptsAuth := bind.NewKeyedTransactor(key)
	transactOptsAuth.Nonce = big.NewInt(int64(nonce))
	transactOptsAuth.Value = big.NewInt(0) // in wei
	transactOptsAuth.GasLimit = GasLimit
	transactOptsAuth.GasPrice = gasPrice

	// All ProphecyClaims are made to the CosmosBridge contract
	var targetContract ContractRegistry

	switch event {
	case events.MsgBurn, events.MsgLock:
		targetContract = CosmosBridge
	case events.LogNewProphecyClaim:
		targetContract = Oracle
	}

	// Get the specific contract's address
	target, err := GetAddressFromBridgeRegistry(client, contractAddress, targetContract)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nFetching CosmosBridge contract...")

	// Initialize CosmosBridge instance
	cosmosBridgeInstance, err := cosmosBridge.NewCosmosBridge(target, client)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Sending new ProphecyClaim to CosmosBridge...")
	tx, err := cosmosBridgeInstance.NewProphecyClaim(transactOptsAuth, uint8(claim.ClaimType), claim.CosmosSender, claim.EthereumReceiver, claim.TokenContractAddress, claim.Symbol, claim.Amount)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nNewProphecyClaim tx hash:", tx.Hash().Hex())

	// Get the transaction receipt
	receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		log.Fatal(err)
	}

	// Report tx status
	switch receipt.Status {
	case 0:
		fmt.Println("Status: 0 - Failed")
	case 1:
		fmt.Println("Status: 1 - Successful")
	}

	return nil
}

// RelayOracleClaimToEthereum : relays the provided OracleClaim to the Ethereum network
func RelayOracleClaimToEthereum(provider string, contractAddress common.Address, event events.Event, claim OracleClaim) error {

	client, err := ethclient.Dial(provider)
	if err != nil {
		log.Fatal(err)
	}

	// Load the validator's private key
	key, err := LoadPrivateKey()
	if err != nil {
		log.Fatal(err)
	}

	// Load the validator's address
	sender, err := LoadSender()
	if err != nil {
		log.Fatal(err)
	}

	nonce, err := client.PendingNonceAt(context.Background(), sender)
	if err != nil {
		log.Fatal(err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Set up TransactOpts auth's tx signature authorization
	transactOptsAuth := bind.NewKeyedTransactor(key)
	transactOptsAuth.Nonce = big.NewInt(int64(nonce))
	transactOptsAuth.Value = big.NewInt(0) // in wei
	transactOptsAuth.GasLimit = GasLimit
	transactOptsAuth.GasPrice = gasPrice

	// All ProphecyClaims are made to the CosmosBridge contract
	var targetContract ContractRegistry

	switch event {
	case events.MsgBurn, events.MsgLock:
		targetContract = CosmosBridge
	case events.LogNewProphecyClaim:
		targetContract = Oracle
	}

	// Get the specific contract's address
	target, err := GetAddressFromBridgeRegistry(client, contractAddress, targetContract)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Oracle instance
	fmt.Println("\nFetching Oracle contract...")
	oracleInstance, err := oracle.NewOracle(target, client)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Sending new OracleClaim to Oracle...")
	tx, err := oracleInstance.NewOracleClaim(transactOptsAuth, claim.ProphecyID, claim.Message, claim.Signature)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nNewOracleClaim tx hash:", tx.Hash().Hex())
	// Get the transaction receipt
	receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		log.Fatal(err)
	}

	// Report tx status
	switch receipt.Status {
	case 0:
		fmt.Println("Status: 0 - Failed")
	case 1:
		fmt.Println("Status: 1 - Successful")
	}

	return nil
}

// TODO: Refactor relay functions to use getRelayConfig()
func getRelayConfig(provider string, registry common.Address, event events.Event) (client *ethclient.Client, auth *bind.TransactOpts, target common.Address) {
	// Start Ethereum client
	client, err := ethclient.Dial(provider)
	if err != nil {
		log.Fatal(err)
	}

	// Load the validator's private key
	key, err := LoadPrivateKey()
	if err != nil {
		log.Fatal(err)
	}

	// Load the validator's address
	sender, err := LoadSender()
	if err != nil {
		log.Fatal(err)
	}

	nonce, err := client.PendingNonceAt(context.Background(), sender)
	if err != nil {
		log.Fatal(err)
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Set up TransactOpts auth's tx signature authorization
	transactOptsAuth := bind.NewKeyedTransactor(key)
	transactOptsAuth.Nonce = big.NewInt(int64(nonce))
	transactOptsAuth.Value = big.NewInt(0) // in wei
	transactOptsAuth.GasLimit = GasLimit
	transactOptsAuth.GasPrice = gasPrice

	// All ProphecyClaims are made to the CosmosBridge contract
	var targetContract ContractRegistry

	switch event {
	case events.MsgBurn, events.MsgLock:
		targetContract = CosmosBridge
	case events.LogNewProphecyClaim:
		targetContract = Oracle
	}

	// Get the specific contract's address
	target, err = GetAddressFromBridgeRegistry(client, registry, targetContract)
	if err != nil {
		log.Fatal(err)
	}

	return client, auth, target
}

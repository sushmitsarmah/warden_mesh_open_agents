package inft

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	mu         sync.Mutex
	transactor *bind.TransactOpts
	contract   *SwarmINFT
	client     *ethclient.Client
}

func NewClient(rpcURL, contractAddr, privateKeyHex string) (*Client, error) {
	cl, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}
	pk, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, err
	}
	chainID, err := cl.ChainID(context.Background())
	if err != nil {
		return nil, err
	}
	transactor, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		return nil, err
	}
	addr := common.HexToAddress(contractAddr)
	// Placeholder: generate bindings from ABI
	_ = addr
	return &Client{
		transactor: transactor,
		client:     cl,
	}, nil
}

func (c *Client) RecordDisclosure(ctx context.Context, tokenID, bountyUsd int64, memoryDelta [32]byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// c.contract.RecordDisclosure(c.transactor, big.NewInt(tokenID), big.NewInt(bountyUsd), memoryDelta)
	return nil
}

func (c *Client) IsPaused() (bool, error) {
	// return c.contract.State(nil, big.NewInt(1))
	return false, nil
}

func (c *Client) IsAuthorized(protocol string) bool {
	// return c.contract.AuthorizedProtocols(nil, common.HexToAddress(protocol))
	return false
}

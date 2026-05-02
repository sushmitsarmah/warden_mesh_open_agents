package inft

import (
	"context"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	mu         sync.Mutex
	transactor *bind.TransactOpts
	contract   *Inft
	client     *ethclient.Client
}

func NewClient(rpcURL, contractAddr, privateKeyHex string) (*Client, error) {
	cl, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial rpc: %w", err)
	}
	pk, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("parse key: %w", err)
	}
	chainID, err := cl.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("chain id: %w", err)
	}
	transactor, err := bind.NewKeyedTransactorWithChainID(pk, chainID)
	if err != nil {
		return nil, fmt.Errorf("transactor: %w", err)
	}

	addr := common.HexToAddress(contractAddr)
	contract, err := NewInft(addr, cl)
	if err != nil {
		return nil, fmt.Errorf("bind contract: %w", err)
	}

	return &Client{
		transactor: transactor,
		contract:   contract,
		client:     cl,
	}, nil
}

func (c *Client) RecordDisclosure(ctx context.Context, tokenID, bountyUsd int64, memoryDelta [32]byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	_, err := c.contract.RecordDisclosure(c.transactor, big.NewInt(tokenID), big.NewInt(bountyUsd), memoryDelta)
	return err
}

func (c *Client) IsPaused(tokenID int64) (bool, error) {
	s, err := c.contract.State(&bind.CallOpts{}, big.NewInt(tokenID))
	if err != nil {
		return false, err
	}
	return s.Paused, nil
}

func (c *Client) IsAuthorized(protocol string) (bool, error) {
	addr := common.HexToAddress(protocol)
	return c.contract.AuthorizedProtocols(&bind.CallOpts{}, addr)
}

// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package inft

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// InftMetaData contains all meta data concerning the Inft contract.
var InftMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"_multisig\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_orchestrator\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"authorizedProtocols\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"multisig\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"orchestrator\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"recordDisclosure\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"bountyUsd\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"memoryDelta\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setAuthorizedProtocol\",\"inputs\":[{\"name\":\"protocol\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"ok\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setPaused\",\"inputs\":[{\"name\":\"_paused\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"state\",\"inputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"disclosuresCount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"cumulativeBountyUsd\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"memoryPointer\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"paused\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"AuthorizedProtocolSet\",\"inputs\":[{\"name\":\"protocol\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"ok\",\"type\":\"bool\",\"indexed\":false,\"internalType\":\"bool\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DisclosureRecorded\",\"inputs\":[{\"name\":\"tokenId\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"bountyUsd\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"memoryDelta\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Paused\",\"inputs\":[{\"name\":\"paused\",\"type\":\"bool\",\"indexed\":false,\"internalType\":\"bool\"}],\"anonymous\":false}]",
}

// InftABI is the input ABI used to generate the binding from.
// Deprecated: Use InftMetaData.ABI instead.
var InftABI = InftMetaData.ABI

// Inft is an auto generated Go binding around an Ethereum contract.
type Inft struct {
	InftCaller     // Read-only binding to the contract
	InftTransactor // Write-only binding to the contract
	InftFilterer   // Log filterer for contract events
}

// InftCaller is an auto generated read-only Go binding around an Ethereum contract.
type InftCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InftTransactor is an auto generated write-only Go binding around an Ethereum contract.
type InftTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InftFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InftFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InftSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InftSession struct {
	Contract     *Inft             // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// InftCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InftCallerSession struct {
	Contract *InftCaller   // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// InftTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InftTransactorSession struct {
	Contract     *InftTransactor   // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// InftRaw is an auto generated low-level Go binding around an Ethereum contract.
type InftRaw struct {
	Contract *Inft // Generic contract binding to access the raw methods on
}

// InftCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InftCallerRaw struct {
	Contract *InftCaller // Generic read-only contract binding to access the raw methods on
}

// InftTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InftTransactorRaw struct {
	Contract *InftTransactor // Generic write-only contract binding to access the raw methods on
}

// NewInft creates a new instance of Inft, bound to a specific deployed contract.
func NewInft(address common.Address, backend bind.ContractBackend) (*Inft, error) {
	contract, err := bindInft(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Inft{InftCaller: InftCaller{contract: contract}, InftTransactor: InftTransactor{contract: contract}, InftFilterer: InftFilterer{contract: contract}}, nil
}

// NewInftCaller creates a new read-only instance of Inft, bound to a specific deployed contract.
func NewInftCaller(address common.Address, caller bind.ContractCaller) (*InftCaller, error) {
	contract, err := bindInft(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InftCaller{contract: contract}, nil
}

// NewInftTransactor creates a new write-only instance of Inft, bound to a specific deployed contract.
func NewInftTransactor(address common.Address, transactor bind.ContractTransactor) (*InftTransactor, error) {
	contract, err := bindInft(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InftTransactor{contract: contract}, nil
}

// NewInftFilterer creates a new log filterer instance of Inft, bound to a specific deployed contract.
func NewInftFilterer(address common.Address, filterer bind.ContractFilterer) (*InftFilterer, error) {
	contract, err := bindInft(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InftFilterer{contract: contract}, nil
}

// bindInft binds a generic wrapper to an already deployed contract.
func bindInft(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := InftMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Inft *InftRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Inft.Contract.InftCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Inft *InftRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Inft.Contract.InftTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Inft *InftRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Inft.Contract.InftTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Inft *InftCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Inft.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Inft *InftTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Inft.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Inft *InftTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Inft.Contract.contract.Transact(opts, method, params...)
}

// AuthorizedProtocols is a free data retrieval call binding the contract method 0x40c340ec.
//
// Solidity: function authorizedProtocols(address ) view returns(bool)
func (_Inft *InftCaller) AuthorizedProtocols(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _Inft.contract.Call(opts, &out, "authorizedProtocols", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// AuthorizedProtocols is a free data retrieval call binding the contract method 0x40c340ec.
//
// Solidity: function authorizedProtocols(address ) view returns(bool)
func (_Inft *InftSession) AuthorizedProtocols(arg0 common.Address) (bool, error) {
	return _Inft.Contract.AuthorizedProtocols(&_Inft.CallOpts, arg0)
}

// AuthorizedProtocols is a free data retrieval call binding the contract method 0x40c340ec.
//
// Solidity: function authorizedProtocols(address ) view returns(bool)
func (_Inft *InftCallerSession) AuthorizedProtocols(arg0 common.Address) (bool, error) {
	return _Inft.Contract.AuthorizedProtocols(&_Inft.CallOpts, arg0)
}

// Multisig is a free data retrieval call binding the contract method 0x4783c35b.
//
// Solidity: function multisig() view returns(address)
func (_Inft *InftCaller) Multisig(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Inft.contract.Call(opts, &out, "multisig")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Multisig is a free data retrieval call binding the contract method 0x4783c35b.
//
// Solidity: function multisig() view returns(address)
func (_Inft *InftSession) Multisig() (common.Address, error) {
	return _Inft.Contract.Multisig(&_Inft.CallOpts)
}

// Multisig is a free data retrieval call binding the contract method 0x4783c35b.
//
// Solidity: function multisig() view returns(address)
func (_Inft *InftCallerSession) Multisig() (common.Address, error) {
	return _Inft.Contract.Multisig(&_Inft.CallOpts)
}

// Orchestrator is a free data retrieval call binding the contract method 0xb74795d9.
//
// Solidity: function orchestrator() view returns(address)
func (_Inft *InftCaller) Orchestrator(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Inft.contract.Call(opts, &out, "orchestrator")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Orchestrator is a free data retrieval call binding the contract method 0xb74795d9.
//
// Solidity: function orchestrator() view returns(address)
func (_Inft *InftSession) Orchestrator() (common.Address, error) {
	return _Inft.Contract.Orchestrator(&_Inft.CallOpts)
}

// Orchestrator is a free data retrieval call binding the contract method 0xb74795d9.
//
// Solidity: function orchestrator() view returns(address)
func (_Inft *InftCallerSession) Orchestrator() (common.Address, error) {
	return _Inft.Contract.Orchestrator(&_Inft.CallOpts)
}

// State is a free data retrieval call binding the contract method 0x3e4f49e6.
//
// Solidity: function state(uint256 ) view returns(uint256 disclosuresCount, uint256 cumulativeBountyUsd, bytes32 memoryPointer, bool paused)
func (_Inft *InftCaller) State(opts *bind.CallOpts, arg0 *big.Int) (struct {
	DisclosuresCount    *big.Int
	CumulativeBountyUsd *big.Int
	MemoryPointer       [32]byte
	Paused              bool
}, error) {
	var out []interface{}
	err := _Inft.contract.Call(opts, &out, "state", arg0)

	outstruct := new(struct {
		DisclosuresCount    *big.Int
		CumulativeBountyUsd *big.Int
		MemoryPointer       [32]byte
		Paused              bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.DisclosuresCount = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.CumulativeBountyUsd = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.MemoryPointer = *abi.ConvertType(out[2], new([32]byte)).(*[32]byte)
	outstruct.Paused = *abi.ConvertType(out[3], new(bool)).(*bool)

	return *outstruct, err

}

// State is a free data retrieval call binding the contract method 0x3e4f49e6.
//
// Solidity: function state(uint256 ) view returns(uint256 disclosuresCount, uint256 cumulativeBountyUsd, bytes32 memoryPointer, bool paused)
func (_Inft *InftSession) State(arg0 *big.Int) (struct {
	DisclosuresCount    *big.Int
	CumulativeBountyUsd *big.Int
	MemoryPointer       [32]byte
	Paused              bool
}, error) {
	return _Inft.Contract.State(&_Inft.CallOpts, arg0)
}

// State is a free data retrieval call binding the contract method 0x3e4f49e6.
//
// Solidity: function state(uint256 ) view returns(uint256 disclosuresCount, uint256 cumulativeBountyUsd, bytes32 memoryPointer, bool paused)
func (_Inft *InftCallerSession) State(arg0 *big.Int) (struct {
	DisclosuresCount    *big.Int
	CumulativeBountyUsd *big.Int
	MemoryPointer       [32]byte
	Paused              bool
}, error) {
	return _Inft.Contract.State(&_Inft.CallOpts, arg0)
}

// RecordDisclosure is a paid mutator transaction binding the contract method 0x7e132d46.
//
// Solidity: function recordDisclosure(uint256 tokenId, uint256 bountyUsd, bytes32 memoryDelta) returns()
func (_Inft *InftTransactor) RecordDisclosure(opts *bind.TransactOpts, tokenId *big.Int, bountyUsd *big.Int, memoryDelta [32]byte) (*types.Transaction, error) {
	return _Inft.contract.Transact(opts, "recordDisclosure", tokenId, bountyUsd, memoryDelta)
}

// RecordDisclosure is a paid mutator transaction binding the contract method 0x7e132d46.
//
// Solidity: function recordDisclosure(uint256 tokenId, uint256 bountyUsd, bytes32 memoryDelta) returns()
func (_Inft *InftSession) RecordDisclosure(tokenId *big.Int, bountyUsd *big.Int, memoryDelta [32]byte) (*types.Transaction, error) {
	return _Inft.Contract.RecordDisclosure(&_Inft.TransactOpts, tokenId, bountyUsd, memoryDelta)
}

// RecordDisclosure is a paid mutator transaction binding the contract method 0x7e132d46.
//
// Solidity: function recordDisclosure(uint256 tokenId, uint256 bountyUsd, bytes32 memoryDelta) returns()
func (_Inft *InftTransactorSession) RecordDisclosure(tokenId *big.Int, bountyUsd *big.Int, memoryDelta [32]byte) (*types.Transaction, error) {
	return _Inft.Contract.RecordDisclosure(&_Inft.TransactOpts, tokenId, bountyUsd, memoryDelta)
}

// SetAuthorizedProtocol is a paid mutator transaction binding the contract method 0x65334ac0.
//
// Solidity: function setAuthorizedProtocol(address protocol, bool ok) returns()
func (_Inft *InftTransactor) SetAuthorizedProtocol(opts *bind.TransactOpts, protocol common.Address, ok bool) (*types.Transaction, error) {
	return _Inft.contract.Transact(opts, "setAuthorizedProtocol", protocol, ok)
}

// SetAuthorizedProtocol is a paid mutator transaction binding the contract method 0x65334ac0.
//
// Solidity: function setAuthorizedProtocol(address protocol, bool ok) returns()
func (_Inft *InftSession) SetAuthorizedProtocol(protocol common.Address, ok bool) (*types.Transaction, error) {
	return _Inft.Contract.SetAuthorizedProtocol(&_Inft.TransactOpts, protocol, ok)
}

// SetAuthorizedProtocol is a paid mutator transaction binding the contract method 0x65334ac0.
//
// Solidity: function setAuthorizedProtocol(address protocol, bool ok) returns()
func (_Inft *InftTransactorSession) SetAuthorizedProtocol(protocol common.Address, ok bool) (*types.Transaction, error) {
	return _Inft.Contract.SetAuthorizedProtocol(&_Inft.TransactOpts, protocol, ok)
}

// SetPaused is a paid mutator transaction binding the contract method 0x16c38b3c.
//
// Solidity: function setPaused(bool _paused) returns()
func (_Inft *InftTransactor) SetPaused(opts *bind.TransactOpts, _paused bool) (*types.Transaction, error) {
	return _Inft.contract.Transact(opts, "setPaused", _paused)
}

// SetPaused is a paid mutator transaction binding the contract method 0x16c38b3c.
//
// Solidity: function setPaused(bool _paused) returns()
func (_Inft *InftSession) SetPaused(_paused bool) (*types.Transaction, error) {
	return _Inft.Contract.SetPaused(&_Inft.TransactOpts, _paused)
}

// SetPaused is a paid mutator transaction binding the contract method 0x16c38b3c.
//
// Solidity: function setPaused(bool _paused) returns()
func (_Inft *InftTransactorSession) SetPaused(_paused bool) (*types.Transaction, error) {
	return _Inft.Contract.SetPaused(&_Inft.TransactOpts, _paused)
}

// InftAuthorizedProtocolSetIterator is returned from FilterAuthorizedProtocolSet and is used to iterate over the raw logs and unpacked data for AuthorizedProtocolSet events raised by the Inft contract.
type InftAuthorizedProtocolSetIterator struct {
	Event *InftAuthorizedProtocolSet // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *InftAuthorizedProtocolSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InftAuthorizedProtocolSet)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(InftAuthorizedProtocolSet)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *InftAuthorizedProtocolSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InftAuthorizedProtocolSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InftAuthorizedProtocolSet represents a AuthorizedProtocolSet event raised by the Inft contract.
type InftAuthorizedProtocolSet struct {
	Protocol common.Address
	Ok       bool
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterAuthorizedProtocolSet is a free log retrieval operation binding the contract event 0x8ab4f81ae3317d1dc7b058c78a56ed65f6aefb70b3e518573fd7d0758df5280c.
//
// Solidity: event AuthorizedProtocolSet(address protocol, bool ok)
func (_Inft *InftFilterer) FilterAuthorizedProtocolSet(opts *bind.FilterOpts) (*InftAuthorizedProtocolSetIterator, error) {

	logs, sub, err := _Inft.contract.FilterLogs(opts, "AuthorizedProtocolSet")
	if err != nil {
		return nil, err
	}
	return &InftAuthorizedProtocolSetIterator{contract: _Inft.contract, event: "AuthorizedProtocolSet", logs: logs, sub: sub}, nil
}

// WatchAuthorizedProtocolSet is a free log subscription operation binding the contract event 0x8ab4f81ae3317d1dc7b058c78a56ed65f6aefb70b3e518573fd7d0758df5280c.
//
// Solidity: event AuthorizedProtocolSet(address protocol, bool ok)
func (_Inft *InftFilterer) WatchAuthorizedProtocolSet(opts *bind.WatchOpts, sink chan<- *InftAuthorizedProtocolSet) (event.Subscription, error) {

	logs, sub, err := _Inft.contract.WatchLogs(opts, "AuthorizedProtocolSet")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InftAuthorizedProtocolSet)
				if err := _Inft.contract.UnpackLog(event, "AuthorizedProtocolSet", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAuthorizedProtocolSet is a log parse operation binding the contract event 0x8ab4f81ae3317d1dc7b058c78a56ed65f6aefb70b3e518573fd7d0758df5280c.
//
// Solidity: event AuthorizedProtocolSet(address protocol, bool ok)
func (_Inft *InftFilterer) ParseAuthorizedProtocolSet(log types.Log) (*InftAuthorizedProtocolSet, error) {
	event := new(InftAuthorizedProtocolSet)
	if err := _Inft.contract.UnpackLog(event, "AuthorizedProtocolSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InftDisclosureRecordedIterator is returned from FilterDisclosureRecorded and is used to iterate over the raw logs and unpacked data for DisclosureRecorded events raised by the Inft contract.
type InftDisclosureRecordedIterator struct {
	Event *InftDisclosureRecorded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *InftDisclosureRecordedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InftDisclosureRecorded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(InftDisclosureRecorded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *InftDisclosureRecordedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InftDisclosureRecordedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InftDisclosureRecorded represents a DisclosureRecorded event raised by the Inft contract.
type InftDisclosureRecorded struct {
	TokenId     *big.Int
	BountyUsd   *big.Int
	MemoryDelta [32]byte
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterDisclosureRecorded is a free log retrieval operation binding the contract event 0x7bf4a5cb4d304f355f0c28ac116e772d966c14c42f5b2b8ebab2e65f1d01bc0c.
//
// Solidity: event DisclosureRecorded(uint256 tokenId, uint256 bountyUsd, bytes32 memoryDelta)
func (_Inft *InftFilterer) FilterDisclosureRecorded(opts *bind.FilterOpts) (*InftDisclosureRecordedIterator, error) {

	logs, sub, err := _Inft.contract.FilterLogs(opts, "DisclosureRecorded")
	if err != nil {
		return nil, err
	}
	return &InftDisclosureRecordedIterator{contract: _Inft.contract, event: "DisclosureRecorded", logs: logs, sub: sub}, nil
}

// WatchDisclosureRecorded is a free log subscription operation binding the contract event 0x7bf4a5cb4d304f355f0c28ac116e772d966c14c42f5b2b8ebab2e65f1d01bc0c.
//
// Solidity: event DisclosureRecorded(uint256 tokenId, uint256 bountyUsd, bytes32 memoryDelta)
func (_Inft *InftFilterer) WatchDisclosureRecorded(opts *bind.WatchOpts, sink chan<- *InftDisclosureRecorded) (event.Subscription, error) {

	logs, sub, err := _Inft.contract.WatchLogs(opts, "DisclosureRecorded")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InftDisclosureRecorded)
				if err := _Inft.contract.UnpackLog(event, "DisclosureRecorded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseDisclosureRecorded is a log parse operation binding the contract event 0x7bf4a5cb4d304f355f0c28ac116e772d966c14c42f5b2b8ebab2e65f1d01bc0c.
//
// Solidity: event DisclosureRecorded(uint256 tokenId, uint256 bountyUsd, bytes32 memoryDelta)
func (_Inft *InftFilterer) ParseDisclosureRecorded(log types.Log) (*InftDisclosureRecorded, error) {
	event := new(InftDisclosureRecorded)
	if err := _Inft.contract.UnpackLog(event, "DisclosureRecorded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InftPausedIterator is returned from FilterPaused and is used to iterate over the raw logs and unpacked data for Paused events raised by the Inft contract.
type InftPausedIterator struct {
	Event *InftPaused // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *InftPausedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InftPaused)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(InftPaused)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *InftPausedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InftPausedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InftPaused represents a Paused event raised by the Inft contract.
type InftPaused struct {
	Paused bool
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterPaused is a free log retrieval operation binding the contract event 0x0e2fb031ee032dc02d8011dc50b816eb450cf856abd8261680dac74f72165bd2.
//
// Solidity: event Paused(bool paused)
func (_Inft *InftFilterer) FilterPaused(opts *bind.FilterOpts) (*InftPausedIterator, error) {

	logs, sub, err := _Inft.contract.FilterLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return &InftPausedIterator{contract: _Inft.contract, event: "Paused", logs: logs, sub: sub}, nil
}

// WatchPaused is a free log subscription operation binding the contract event 0x0e2fb031ee032dc02d8011dc50b816eb450cf856abd8261680dac74f72165bd2.
//
// Solidity: event Paused(bool paused)
func (_Inft *InftFilterer) WatchPaused(opts *bind.WatchOpts, sink chan<- *InftPaused) (event.Subscription, error) {

	logs, sub, err := _Inft.contract.WatchLogs(opts, "Paused")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InftPaused)
				if err := _Inft.contract.UnpackLog(event, "Paused", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParsePaused is a log parse operation binding the contract event 0x0e2fb031ee032dc02d8011dc50b816eb450cf856abd8261680dac74f72165bd2.
//
// Solidity: event Paused(bool paused)
func (_Inft *InftFilterer) ParsePaused(log types.Log) (*InftPaused, error) {
	event := new(InftPaused)
	if err := _Inft.contract.UnpackLog(event, "Paused", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

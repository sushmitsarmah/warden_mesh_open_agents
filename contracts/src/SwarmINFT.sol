// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

/// @dev The standard ERC-7857 interface for iNFTs
interface IERC7857 {
    event MemoryUpdated(uint256 indexed tokenId, bytes32 newMemoryPointer);

    /// @notice Returns the current memory pointer for an iNFT
    function memoryPointer(uint256 tokenId) external view returns (bytes32);

    /// @notice Updates the memory pointer (restricted to the AI agent/orchestrator)
    function updateMemory(uint256 tokenId, bytes32 memoryDelta) external;
}

contract SwarmINFT is IERC7857 {
    struct SwarmState {
        uint256 disclosuresCount;
        uint256 cumulativeBountyUsd;
        bytes32 memPointer; // Renamed to avoid shadowing the view function
        bool paused;
    }

    mapping(uint256 => SwarmState) public state;
    mapping(address => bool) public authorizedProtocols;
    address public multisig;
    address public orchestrator;

    event DisclosureRecorded(uint256 tokenId, uint256 bountyUsd, bytes32 memoryDelta);
    event AuthorizedProtocolSet(address protocol, bool ok);
    event Paused(bool paused);

    modifier onlyMultisig() {
        require(msg.sender == multisig, "not multisig");
        _;
    }

    modifier onlyOrchestrator() {
        require(msg.sender == orchestrator, "not orchestrator");
        _;
    }

    constructor(address _multisig, address _orchestrator) {
        multisig = _multisig;
        orchestrator = _orchestrator;
    }

    // REQUIRED: EIP-7857 View Function
    function memoryPointer(uint256 tokenId) external view override returns (bytes32) {
        return state[tokenId].memPointer;
    }

    // REQUIRED: EIP-7857 State Update Function
    function updateMemory(uint256 tokenId, bytes32 memoryDelta) external override onlyOrchestrator {
        _updateMemory(tokenId, memoryDelta);
    }

    function recordDisclosure(uint256 tokenId, uint256 bountyUsd, bytes32 memoryDelta) external onlyOrchestrator {
        SwarmState storage s = state[tokenId];
        s.disclosuresCount += 1;
        s.cumulativeBountyUsd += bountyUsd;
        _updateMemory(tokenId, memoryDelta);
        emit DisclosureRecorded(tokenId, bountyUsd, memoryDelta);
    }

    // Internal memory update — avoids external self-call that resets msg.sender
    function _updateMemory(uint256 tokenId, bytes32 memoryDelta) internal {
        state[tokenId].memPointer = memoryDelta;
        emit MemoryUpdated(tokenId, memoryDelta);
    }

    function setAuthorizedProtocol(address protocol, bool ok) external onlyMultisig {
        authorizedProtocols[protocol] = ok;
        emit AuthorizedProtocolSet(protocol, ok);
    }

    function setPaused(bool _paused) external onlyMultisig {
        state[1].paused = _paused;
        emit Paused(_paused);
    }
}
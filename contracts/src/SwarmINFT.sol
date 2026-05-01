// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract SwarmINFT {
    struct SwarmState {
        uint256 disclosuresCount;
        uint256 cumulativeBountyUsd;
        bytes32 memoryPointer;
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

    function recordDisclosure(uint256 tokenId, uint256 bountyUsd, bytes32 memoryDelta) external onlyOrchestrator {
        SwarmState storage s = state[tokenId];
        s.disclosuresCount += 1;
        s.cumulativeBountyUsd += bountyUsd;
        s.memoryPointer = memoryDelta;
        emit DisclosureRecorded(tokenId, bountyUsd, memoryDelta);
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

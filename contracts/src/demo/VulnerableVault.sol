// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

contract VulnerableVault {
    mapping(address => uint256) public balances;

    function deposit() external payable {
        balances[msg.sender] += msg.value;
    }

    function withdraw() external {
        uint256 bal = balances[msg.sender];
        require(bal > 0, "no balance");
        (bool success, ) = msg.sender.call{value: bal}("");
        require(success, "transfer failed");
        // Vulnerability: state update AFTER external call
        balances[msg.sender] = 0;
    }

    receive() external payable {}
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Script.sol";
import "../src/SwarmINFT.sol";

contract DeployScript is Script {
    function run() external {
        uint256 pk = vm.envUint("OG_PRIVATE_KEY");
        address multisig = vm.envOr("MULTISIG_ADDRESS", address(0));
        address orchestrator = vm.envOr("ORCHESTRATOR_ADDRESS", address(0));
        vm.startBroadcast(pk);
        SwarmINFT inft = new SwarmINFT(multisig, orchestrator);
        vm.stopBroadcast();
        // Write deployment address
        string memory json = string.concat('{"address":"', vm.toString(address(inft)), '"}');
        vm.writeFile("deployments/og_testnet.json", json);
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import "forge-std/Test.sol";
import "../src/SwarmINFT.sol";

contract SwarmINFTTest is Test {
    SwarmINFT inft;
    address multisig = address(1);
    address orchestrator = address(2);
    address rando = address(3);

    function setUp() public {
        inft = new SwarmINFT(multisig, orchestrator);
    }

    function testRecordDisclosure() public {
        vm.prank(orchestrator);
        inft.recordDisclosure(1, 5000, keccak256("delta"));
        (uint256 count, uint256 bounty, , ) = inft.state(1);
        assertEq(count, 1);
        assertEq(bounty, 5000);
    }

    function testRecordDisclosureRevertNotOrchestrator() public {
        vm.prank(rando);
        vm.expectRevert("not orchestrator");
        inft.recordDisclosure(1, 5000, keccak256("delta"));
    }

    function testSetAuthorizedProtocol() public {
        vm.prank(multisig);
        inft.setAuthorizedProtocol(address(4), true);
        assertTrue(inft.authorizedProtocols(address(4)));
    }

    function testPause() public {
        vm.prank(multisig);
        inft.setPaused(true);
        (,, , bool paused) = inft.state(1);
        assertTrue(paused);
    }
}

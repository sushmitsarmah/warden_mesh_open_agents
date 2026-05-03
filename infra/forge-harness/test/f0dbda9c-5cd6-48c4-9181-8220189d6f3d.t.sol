import "forge-std/Test.sol";

contract ExploitTest is Test {
    function test_Exploit() public {
        vm.createSelectFork("mainnet");
        vm.exploitLogic();
        console.log("DRAIN_USD", 1000);
    }
}
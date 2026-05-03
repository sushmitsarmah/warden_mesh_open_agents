import "forge-std/Test.sol";

contract ExploitTest is Test {
    function test_Exploit() public {
        vm.createSelectFork("mainnet");
        // exploit logic here
        vm.deposit(1000 ether);
        vm.call{value: 1000, gas: 20000} ("DRAIN_USD", 1000);
    }
}
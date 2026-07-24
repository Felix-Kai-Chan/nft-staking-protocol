// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "forge-std/Script.sol";
import "../src/Staking.sol";
import "../src/mocks/MockERC20.sol";

contract DeployStaking is Script {
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("DEPLOYER_PRIVATE_KEY");
        vm.startBroadcast(deployerPrivateKey);

        // 部署代币
        MockERC20 stakeToken = new MockERC20("Stake Token", "STK");
        MockERC20 rewardToken = new MockERC20("Reward Token", "RWD");

        // 给部署者铸币
        address deployer = vm.addr(deployerPrivateKey);
        stakeToken.mint(deployer, 10000 ether);
        rewardToken.mint(deployer, 10000 ether);

        // 部署 Staking
        Staking staking = new Staking(
            address(stakeToken),
            address(rewardToken),
            1 ether
        );

        // 转移奖励代币到 Staking 合约
        rewardToken.transfer(address(staking), 10000 ether);

        vm.stopBroadcast();

        console.log("Staking deployed at:", address(staking));
        console.log("Stake Token:", address(stakeToken));
        console.log("Reward Token:", address(rewardToken));
    }
}
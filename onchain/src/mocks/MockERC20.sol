// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

/**
 * @title MockERC20
 * @notice 用于测试的简单 ERC20 代币，支持无权限铸造
 */
contract MockERC20 is ERC20 {
    // 构造函数：设置代币名称和符号
    constructor(string memory name, string memory symbol) ERC20(name, symbol) {}

    /**
     * @dev 铸造代币
     * @param to 接收地址
     * @param amount 数量
     * @notice 这是一个公开函数，仅用于测试环境！
     */
    function mint(address to, uint256 amount) external {
        _mint(to, amount);
    }

    /**
     * @dev 销毁代币 (可选)
     */
    function burn(address from, uint256 amount) external {
        _burn(from, amount);
    }
}
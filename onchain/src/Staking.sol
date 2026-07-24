// SPDX-License-Identifier: MIT
pragma solidity ^0.8.28;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";

/**
 * @title Staking
 * @notice 用户质押 ERC20 代币，按时间线性释放奖励
 * @dev 奖励代币和质押代币可以是同一个，也可以是不同的
 */
contract Staking is Ownable, ReentrancyGuard {
    using SafeERC20 for IERC20;

    // ========== 状态变量 ==========

    IERC20 public stakeToken;      // 质押代币
    IERC20 public rewardToken;     // 奖励代币
    uint256 public rewardRate;     // 每秒奖励数量（18 位精度）
    uint256 public lastUpdateTime; // 上次更新时间
    uint256 public rewardPerTokenStored; // 累计每份质押的奖励

    uint256 public totalStaked;    // 总质押量

    // 用户质押信息
    struct UserInfo {
        uint256 amount;            // 质押数量
        uint256 rewardPerTokenPaid; // 已结算的奖励
        uint256 rewards;           // 待领取奖励
    }

    mapping(address => UserInfo) public users;

    // ========== 事件 (Events) ==========

    /// @notice 用户质押代币时触发
    /// @dev user 和 amount 均被索引，方便后端按用户或金额范围查询
    /// @param user 质押者地址
    /// @param amount 质押的代币数量
    event Staked(address indexed user, uint256 indexed amount);

    /// @notice 用户提取质押代币时触发
    /// @param user 提取者地址
    /// @param amount 提取的代币数量
    event Withdrawn(address indexed user, uint256 indexed amount);

    /// @notice 用户领取奖励时触发
    /// @param user 领取者地址
    /// @param reward 领取的奖励数量
    event Claimed(address indexed user, uint256 indexed reward);

    /// @notice 管理员更新奖励速率时触发
    /// @param newRate 新的每秒奖励数量
    event RewardRateUpdated(uint256 indexed newRate);

    // ========== 构造函数 ==========

    constructor(
        address _stakeToken,
        address _rewardToken,
        uint256 _rewardRate
    ) Ownable(msg.sender) {
        stakeToken = IERC20(_stakeToken);
        rewardToken = IERC20(_rewardToken);
        rewardRate = _rewardRate;
        lastUpdateTime = block.timestamp;
    }

    // ========== 修改器 ==========

    modifier updateReward(address account) {
        rewardPerTokenStored = rewardPerToken();
        lastUpdateTime = block.timestamp;
        if (account != address(0)) {
            users[account].rewards = earned(account);
            users[account].rewardPerTokenPaid = rewardPerTokenStored;
        }
        _;
    }

    // ========== 核心函数 ==========

    /**
     * @dev 用户质押代币
     */
    function stake(uint256 amount) external nonReentrant updateReward(msg.sender) {
        require(amount > 0, "Cannot stake 0");
        require(stakeToken.balanceOf(msg.sender) >= amount, "Insufficient balance");
        require(stakeToken.allowance(msg.sender, address(this)) >= amount, "Insufficient allowance");

        totalStaked += amount;
        users[msg.sender].amount += amount;

        stakeToken.safeTransferFrom(msg.sender, address(this), amount);

        emit Staked(msg.sender, amount);
    }

    /**
     * @dev 用户取回质押代币
     */
    function withdraw(uint256 amount) public nonReentrant updateReward(msg.sender) {
        require(amount > 0, "Cannot withdraw 0");
        require(users[msg.sender].amount >= amount, "Insufficient staked amount");

        users[msg.sender].amount -= amount;
        totalStaked -= amount;

        stakeToken.safeTransfer(msg.sender, amount);

        emit Withdrawn(msg.sender, amount);
    }

    /**
     * @dev 用户领取奖励
     */
    function claimReward() public nonReentrant updateReward(msg.sender) {
        uint256 reward = users[msg.sender].rewards;
        if (reward > 0) {
            users[msg.sender].rewards = 0;
            rewardToken.safeTransfer(msg.sender, reward);
            emit Claimed(msg.sender, reward);
        }
    }

    /**
     * @dev 取出全部质押并领取奖励
     */
    function exit() external {
        withdraw(users[msg.sender].amount);
        claimReward();
    }

    // ========== 视图函数 ==========

    /**
     * @dev 计算当前每份质押的累计奖励
     */
    function rewardPerToken() public view returns (uint256) {
        if (totalStaked == 0) {
            return rewardPerTokenStored;
        }
        return rewardPerTokenStored + ((block.timestamp - lastUpdateTime) * rewardRate * 1e18) / totalStaked;
    }

    /**
     * @dev 计算用户当前待领取奖励
     */
    function earned(address account) public view returns (uint256) {
        return ((users[account].amount * (rewardPerToken() - users[account].rewardPerTokenPaid)) / 1e18) + users[account].rewards;
    }

    /**
     * @dev 查询用户质押信息
     */
    function getUserInfo(address account) external view returns (UserInfo memory) {
        return users[account];
    }

    // ========== 管理员函数 ==========

    /**
     * @dev 更新奖励速率（仅管理员）
     */
    function setRewardRate(uint256 _rewardRate) external onlyOwner updateReward(address(0)) {
        rewardRate = _rewardRate;
        emit RewardRateUpdated(_rewardRate);
    }

    /**
     * @dev 紧急提取合约中的奖励代币（仅管理员）
     */
    function emergencyWithdrawReward(uint256 amount) external onlyOwner {
        rewardToken.safeTransfer(owner(), amount);
    }
}
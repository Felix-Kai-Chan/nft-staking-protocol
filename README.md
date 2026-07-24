# 🪙 NFT Staking Protocol

一个基于 **Solidity + Go** 构建的生产级 Web3 质押协议。

该项目包含：

- ⚡ Solidity 智能合约
- 🧪 Foundry 测试框架
- 🔧 Go 后端服务
- 📡 Ethereum 事件索引器
- 🗄️ MySQL 数据持久化
- 🌐 RESTful API

---

## 📋 目录

- [技术栈](#技术栈)
- [系统架构](#系统架构)
- [智能合约](#智能合约)
- [合约测试](#合约测试)
- [项目结构](#项目结构)
- [数据库设计](#数据库设计)
- [REST API](#rest-api)
- [快速启动](#快速启动)
- [工作流程](#工作流程)
- [后续计划](#后续计划)

---

## 🛠️ 技术栈

| 层级 | 技术 |
|------|------|
| 区块链 | Solidity, Foundry, OpenZeppelin |
| 后端 | Go 1.25.7, Gin, GORM |
| 数据库 | MySQL |
| Ethereum | go-ethereum, RPC, 事件索引, ABI 绑定 |

---

## 🏗️ 系统架构

```
Blockchain
    │
    ▼
Smart Contract
    │
    │ Events
    ▼
Indexer (Go)
    │
    ▼
MySQL
    │
    ▼
REST API
```

---

## 📄 智能合约

**文件：** `Staking.sol`

基于 OpenZeppelin 标准实现，包含：

- ERC20 代币质押
- 线性奖励分发
- 用户奖励记账
- 事件日志推送
- 安全机制（SafeERC20 + Ownable + ReentrancyGuard）

**支持的功能：**

| 功能 | 说明 |
|------|------|
| 质押 | 质押 ERC20 代币获取奖励 |
| 提现 | 提取已质押的代币 |
| 领取奖励 | 领取累积的质押奖励 |
| 查询信息 | 查询用户质押和奖励信息 |
| 更新费率 | 更新奖励速率（仅管理员） |
| 紧急提取 | 紧急提取奖励代币（仅管理员） |

**合约事件：**

```solidity
event Staked(address indexed user, uint256 indexed amount);
event Withdrawn(address indexed user, uint256 indexed amount);
event Claimed(address indexed user, uint256 indexed reward);
event RewardRateUpdated(uint256 indexed newRate);
```

---

## 🧪 合约测试

**测试框架：** Foundry

**运行测试：**
```bash
cd onchain
forge test
```

**覆盖率：**
```bash
forge coverage
```

**测试结果：** 16 个测试全部通过

**覆盖场景：** 质押、提现、领取奖励、退出、奖励计算、管理员权限、回退场景、Fuzz 测试、部署

**本地部署：**
```bash
cd onchain
forge script script/DeployStaking.s.sol \
  --rpc-url http://localhost:8545 \
  --broadcast \
  --private-key <PRIVATE_KEY>
```

---

## 📁 项目结构

```
nft-staking-protocol
├── onchain/
│   ├── src/
│   │   ├── Staking.sol
│   │   └── mocks/
│   │       └── MockERC20.sol
│   ├── test/
│   │   └── Staking.t.sol
│   └── script/
│       └── DeployStaking.s.sol
│
└── offchain/
    ├── cmd/
    │   ├── api/
    │   │   └── main.go
    │   └── indexer/
    │       └── main.go
    ├── internal/
    │   ├── api/
    │   │   ├── handler/
    │   │   ├── service/
    │   │   └── dto/
    │   ├── indexer/
    │   │   ├── listener.go
    │   │   └── repository/
    │   ├── contract/
    │   │   └── staking.go
    │   └── infra/
    │       └── eth/
    └── abi/
        └── Staking.json
```

---

## 🗄️ 数据库设计

### stakes 表（质押记录）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user_address | varchar(42) | 用户地址 |
| amount | varchar(78) | 质押数量 |
| staked_at | uint64 | 质押时间戳 |
| reward_debt | uint64 | 已结算奖励 |
| last_updated | uint64 | 最后更新时间 |

### reward_claims 表（领取记录）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| user | varchar(42) | 用户地址 |
| reward | varchar(78) | 奖励数量 |
| created_at | timestamp | 领取时间戳 |

### sync_status 表（同步状态）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| last_block | uint64 | 最后同步区块 |
| updated_at | int64 | 更新时间戳 |

**数据一致性：**
- 使用 `ON DUPLICATE KEY UPDATE` 防止重复插入
- `uint256` 值使用 `VARCHAR(78)` 存储，防止溢出
- `sync_status` 记录检查点，支持重启恢复

---

## 🌐 REST API

### 健康检查
```
GET /health
```

### 查询质押信息
```
GET /api/v1/stake/:address
```

示例：
```bash
curl http://localhost:8080/api/v1/stake/0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
```

响应：
```json
{
  "address": "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
  "amount": "1000000000000000000000",
  "staked_at": 1784900120
}
```

### 查询领取历史
```
GET /api/v1/stake/:address/claims
```

响应：
```json
{
  "address": "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
  "claims": [
    {
      "ID": 1,
      "User": "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
      "Reward": "317000000000000000000",
      "CreatedAt": "2026-07-24T21:41:10.785+08:00"
    }
  ]
}
```

### 质押
```
POST /api/v1/stake
```

请求：
```json
{
  "amount": "100000000000000000000"
}
```

响应：
```json
{
  "tx_hash": "0x...",
  "amount": "100000000000000000000"
}
```

### 提现
```
POST /api/v1/withdraw
```

请求：
```json
{
  "amount": "200000000000000000000"
}
```

响应：
```json
{
  "tx_hash": "0x...",
  "amount": "200000000000000000000"
}
```

### 领取奖励
```
POST /api/v1/claim
```

请求：
```json
{}
```

响应：
```json
{
  "tx_hash": "0x..."
}
```

---

## 🚀 快速启动

### 1. 启动本地区块链
```bash
anvil --chain-id 31337 --host 0.0.0.0
```

### 2. 部署合约
```bash
cd onchain
forge script script/DeployStaking.s.sol \
  --rpc-url http://localhost:8545 \
  --broadcast \
  --private-key <PRIVATE_KEY>
```

### 3. 启动索引器
```bash
cd offchain
go run cmd/indexer/main.go
```

### 4. 启动 API 服务
```bash
cd offchain
go run cmd/api/main.go
```

服务地址：`http://localhost:8080`

---

## 🔄 工作流程

```
1. 用户授权 ERC20 代币
        │
        ▼
2. 用户调用 stake()
        │
        ▼
3. 合约推送 Staked 事件
        │
        ▼
4. 索引器接收事件
        │
        ▼
5. 数据存入 MySQL
        │
        ▼
6. API 提供查询
```

---

## 📝 后续计划

- Redis 缓存层
- RabbitMQ 消息队列
- 多链支持
- 生产环境部署
- 监控和指标
- 前端 DApp 集成

---

## 👤 作者

Web3 后端工程实践项目。

**关注领域：**
- 智能合约开发
- 区块链事件索引
- 后端架构
- Web3 基础设施
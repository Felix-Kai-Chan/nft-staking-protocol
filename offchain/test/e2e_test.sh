#!/bin/bash

# ============================================
# Staking E2E 测试
# 场景：触发链上 stake 事件 → Indexer → MQ → Consumer → MySQL
# ============================================

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASS=0
FAIL=0

# 断言函数
assert_eq() {
    local actual=$1
    local expected=$2
    local msg=$3
    if [ "$actual" == "$expected" ]; then
        echo -e "${GREEN}✅ PASS: $msg${NC}"
        ((PASS++))
    else
        echo -e "${RED}❌ FAIL: $msg (期望 $expected, 实际 $actual)${NC}"
        ((FAIL++))
    fi
}

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}   Staking E2E 测试${NC}"
echo -e "${YELLOW}========================================${NC}"

# ============================================
# 配置
# ============================================
RPC_URL="http://127.0.0.1:8545"
STAKING_CONTRACT="0xDc64a140Aa3E981100a9becA4E685f962f0cF6C9"
STAKE_TOKEN="0x5FbDB2315678afecb367f032d93F642f64180aa3"
PRIVATE_KEY="ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
USER_ADDRESS="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
STAKE_AMOUNT="100000000000000000000"  # 100 tokens

# MySQL 连接（本地）
MYSQL_CMD="docker exec staking-mysql mysql -uroot -proot -e"
MYSQL_DB="USE staking;"

# ============================================
# 第一步：重置 DB + Cursor
# ============================================
echo -e "\n${YELLOW}📦 1. 重置测试数据...${NC}"

$MYSQL_CMD "$MYSQL_DB TRUNCATE TABLE event_log; TRUNCATE TABLE stakes; TRUNCATE TABLE reward_claims; UPDATE sync_cursor SET last_block = 0 WHERE id = 1;" 2>/dev/null

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ 无法连接 MySQL，请确认本地 MySQL 在 3306 端口运行${NC}"
    exit 1
fi

# 如果没有 cursor 记录，插入一条
$MYSQL_CMD "$MYSQL_DB INSERT IGNORE INTO sync_cursor (id, last_block, updated_at) VALUES (1, 0, 0);" 2>/dev/null

echo -e "${GREEN}✅ 数据已重置${NC}"

# ============================================
# 第二步：确认 anvil + 合约
# ============================================
echo -e "\n${YELLOW}🔗 2. 确认链上状态...${NC}"

BLOCK_BEFORE=$(curl -s -X POST $RPC_URL \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    | jq -r '.result')

if [ -z "$BLOCK_BEFORE" ] || [ "$BLOCK_BEFORE" == "null" ]; then
    echo -e "${RED}❌ anvil 未运行或 RPC 不可达${NC}"
    exit 1
fi

CURRENT_BLOCK=$(printf "%d" $BLOCK_BEFORE)
echo "当前区块: $CURRENT_BLOCK"

# ============================================
# 第三步：approve + stake
# ============================================
echo -e "\n${YELLOW}💰 3. 触发链上 stake...${NC}"

# approve
NO_PROXY=127.0.0.1,localhost no_proxy=127.0.0.1,localhost \
cast send $STAKE_TOKEN \
    "approve(address,uint256)" \
    $STAKING_CONTRACT \
    1000000000000000000000000 \
    --rpc-url $RPC_URL \
    --private-key $PRIVATE_KEY \
    > /dev/null 2>&1

echo "✅ approve 完成"

# 记录 stake 之前的 event_log 数量
EVENT_COUNT_BEFORE=$($MYSQL_CMD "$MYSQL_DB SELECT COUNT(*) FROM event_log;" 2>/dev/null | tail -1)
echo "Stake 前 event_log 数量: $EVENT_COUNT_BEFORE"

# stake
STAKE_TX=$(NO_PROXY=127.0.0.1,localhost no_proxy=127.0.0.1,localhost \
    cast send $STAKING_CONTRACT \
    "stake(uint256)" \
    $STAKE_AMOUNT \
    --rpc-url $RPC_URL \
    --private-key $PRIVATE_KEY \
    --json 2>/dev/null)

TX_HASH=$(echo $STAKE_TX | jq -r '.transactionHash')
echo "Stake 交易哈希: $TX_HASH"

if [ -z "$TX_HASH" ] || [ "$TX_HASH" == "null" ]; then
    echo -e "${RED}❌ stake 失败${NC}"
    exit 1
fi

echo -e "${GREEN}✅ stake 成功${NC}"

# ============================================
# 第四步：等待 Indexer + Consumer 处理
# ============================================
echo -e "\n${YELLOW}⏳ 4. 等待 Indexer + Consumer 处理（最多 15 秒）...${NC}"

MAX_WAIT=15
WAITED=0
EVENT_COUNT_AFTER=$EVENT_COUNT_BEFORE

while [ $WAITED -lt $MAX_WAIT ]; do
    sleep 1
    WAITED=$((WAITED + 1))

    EVENT_COUNT_AFTER=$($MYSQL_CMD "$MYSQL_DB SELECT COUNT(*) FROM event_log WHERE tx_hash = '$TX_HASH';" 2>/dev/null | tail -1)

    if [ "$EVENT_COUNT_AFTER" == "1" ]; then
        echo "✅ 已处理（等待 ${WAITED}s）"
        break
    fi
done

if [ "$EVENT_COUNT_AFTER" != "1" ]; then
    echo -e "${RED}❌ 超时：Indexer + Consumer 未处理事件（${MAX_WAIT}s）${NC}"
fi

# ============================================
# 第五步：验证 DB
# ============================================
echo -e "\n${YELLOW}📊 5. 验证 DB...${NC}"

# 5.1 event_log
EVENT_COUNT=$($MYSQL_CMD "$MYSQL_DB SELECT COUNT(*) FROM event_log WHERE tx_hash = '$TX_HASH';" 2>/dev/null | tail -1)
echo "event_log 记录数: $EVENT_COUNT"
assert_eq "$EVENT_COUNT" "1" "event_log 应该有 1 条（去重表写入）"

# 5.2 stakes
STAKE_COUNT=$($MYSQL_CMD "$MYSQL_DB SELECT COUNT(*) FROM stakes WHERE user_address = '$USER_ADDRESS';" 2>/dev/null | tail -1)
echo "stakes 记录数: $STAKE_COUNT"
assert_eq "$STAKE_COUNT" "1" "stakes 应该有 1 条"

# 5.3 stakes.amount
STAKE_AMOUNT_DB=$($MYSQL_CMD "$MYSQL_DB SELECT amount FROM stakes WHERE user_address = '$USER_ADDRESS';" 2>/dev/null | tail -1)
echo "stakes.amount: $STAKE_AMOUNT_DB"
assert_eq "$STAKE_AMOUNT_DB" "$STAKE_AMOUNT" "stakes.amount 应该等于 $STAKE_AMOUNT"

# 5.4 sync_cursor 是否推进
CURSOR_BLOCK=$($MYSQL_CMD "$MYSQL_DB SELECT last_block FROM sync_cursor WHERE id = 1;" 2>/dev/null | tail -1)
echo "sync_cursor.last_block: $CURSOR_BLOCK"

if [ "$CURSOR_BLOCK" -gt 0 ]; then
    echo -e "${GREEN}✅ PASS: sync_cursor 已推进（$CURSOR_BLOCK）${NC}"
    ((PASS++))
else
    echo -e "${RED}❌ FAIL: sync_cursor 未推进${NC}"
    ((FAIL++))
fi

# ============================================
# 汇总
# ============================================
echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}   测试完成：$PASS PASS / $FAIL FAIL${NC}"
echo -e "${GREEN}========================================${NC}"

if [ $FAIL -gt 0 ]; then
    exit 1
fi
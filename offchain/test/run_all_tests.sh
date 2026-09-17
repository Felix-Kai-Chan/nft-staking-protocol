#!/bin/bash

# ============================================
# Staking 全部测试一键执行
# ============================================

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

cd "$(dirname "$0")/.."

echo -e "${YELLOW}========================================${NC}"
echo -e "${YELLOW}   Staking 全部测试${NC}"
echo -e "${YELLOW}========================================${NC}"

# 1. DAO 单元测试
echo -e "\n${YELLOW}[1/5] DAO 单元测试...${NC}"
go test -v -count=1 ./internal/dao/ -run 'Test'

# 2. API Handler 测试
echo -e "\n${YELLOW}[2/5] API Handler 测试...${NC}"
go test -v -count=1 ./internal/api/handler/

# 3. MQ 集成测试
echo -e "\n${YELLOW}[3/5] MQ 集成测试...${NC}"
go test -v -count=1 ./internal/mq/

# 4. 性能测试
echo -e "\n${YELLOW}[4/5] 性能测试...${NC}"
go test -bench=. -benchtime=2s ./internal/dao/

# 5. E2E 测试
echo -e "\n${YELLOW}[5/5] E2E 测试...${NC}"
chmod +x ./test/e2e_test.sh
./test/e2e_test.sh

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}   ✅ 全部测试通过${NC}"
echo -e "${GREEN}========================================${NC}"
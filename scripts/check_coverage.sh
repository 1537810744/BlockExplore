#!/bin/bash
# 检查测试覆盖率是否达标
# 用法: bash scripts/check_coverage.sh [阈值，默认 40]

set -e

THRESHOLD=${1:-40}

echo "==> 运行测试并生成覆盖率报告..."
go test -coverprofile=coverage.out -covermode=atomic -run=XXX_NONE ./... 2>&1

# 提取总覆盖率（百分比数字）
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

echo "=========================================="
echo "总测试覆盖率: ${COVERAGE}%"
echo "最低要求:     ${THRESHOLD}%"
echo "=========================================="

if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
    echo "FAIL: 测试覆盖率不达标！"
    exit 1
else
    echo "OK: 测试覆盖率达标。"
    exit 0
fi

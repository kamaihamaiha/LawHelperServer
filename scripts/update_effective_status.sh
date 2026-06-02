#!/bin/bash
# 根据 effectDate 和当前日期更新 laws_list 的 effectiveStatus
# effectiveStatus=3 已生效, effectiveStatus=4 尚未生效

DB_PATH="$(cd "$(dirname "$0")/.." && pwd)/data/db/laws.db"

if [ ! -f "$DB_PATH" ]; then
    echo "数据库文件不存在: $DB_PATH"
    exit 1
fi

TODAY=$(date +%Y-%m-%d)
echo "当前日期: $TODAY"

# effectDate <= 今天 但 status=4 → 改为3(已生效)
TO_EFFECTIVE=$(sqlite3 "$DB_PATH" "
    SELECT COUNT(*) FROM laws_list
    WHERE effectDate IS NOT NULL AND effectDate != '' AND effectDate <= '$TODAY'
      AND effectiveStatus = 4;
")

# effectDate > 今天 但 status!=4 → 改为4(尚未生效)
TO_PENDING=$(sqlite3 "$DB_PATH" "
    SELECT COUNT(*) FROM laws_list
    WHERE effectDate IS NOT NULL AND effectDate != '' AND effectDate > '$TODAY'
      AND effectiveStatus != 4;
")

echo "将更新 ${TO_EFFECTIVE} 条记录: 4(尚未生效) → 3(已生效)"
echo "将更新 ${TO_PENDING} 条记录: 其他状态 → 4(尚未生效)"

if [ "$TO_EFFECTIVE" -eq 0 ] && [ "$TO_PENDING" -eq 0 ]; then
    echo "无需更新"
    exit 0
fi

sqlite3 "$DB_PATH" "
    UPDATE laws_list SET effectiveStatus = 3
    WHERE effectDate IS NOT NULL AND effectDate != '' AND effectDate <= '$TODAY'
      AND effectiveStatus = 4;

    UPDATE laws_list SET effectiveStatus = 4
    WHERE effectDate IS NOT NULL AND effectDate != '' AND effectDate > '$TODAY'
      AND effectiveStatus != 4;
"

echo "更新完成"

# 显示更新后的状态分布
echo ""
echo "更新后 effectiveStatus 分布:"
sqlite3 "$DB_PATH" "
    SELECT effectiveStatus, COUNT(*) FROM laws_list GROUP BY effectiveStatus ORDER BY effectiveStatus;
" | while IFS='|' read status count; do
    case $status in
        -1) label="已废止" ;;
        1)  label="部分失效" ;;
        2)  label="尚未生效" ;;
        3)  label="已生效" ;;
        4)  label="尚未生效" ;;
        *)  label="未知" ;;
    esac
    echo "  $status ($label): $count"
done

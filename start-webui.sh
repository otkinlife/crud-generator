#!/bin/bash

# CRUD Generator 启动脚本 - 基于JSON配置的安全版本

# 设置默认配置目录路径
export CONFIG_PATH="${CONFIG_PATH:-./configs}"

echo "Starting CRUD Generator Web Configuration Manager..."
echo "Config directory: $CONFIG_PATH"

# 检查配置目录是否存在
if [ ! -d "$CONFIG_PATH" ]; then
    echo "❌ Error: Config directory not found: $CONFIG_PATH"
    echo "Please create the config directory first."
    exit 1
fi

# 检查数据库配置文件是否存在
DB_CONFIG_FILE="$CONFIG_PATH/db.json"
if [ ! -f "$DB_CONFIG_FILE" ]; then
    echo "❌ Error: Database config file not found: $DB_CONFIG_FILE"
    echo "Please create the database configuration file first."
    exit 1
fi

echo "✅ Database config file found: $DB_CONFIG_FILE"

# 构建并运行Web UI
go run cmd/crud-generator/main.go "$@"
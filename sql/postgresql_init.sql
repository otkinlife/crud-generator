-- CRUD Generator PostgreSQL 数据库初始化脚本

-- 表配置管理表
CREATE TABLE IF NOT EXISTS table_configurations (
    id SERIAL PRIMARY KEY,
    connection_id VARCHAR(100) NOT NULL,
    name VARCHAR(100) NOT NULL,
    table_name VARCHAR(100) NOT NULL,
    create_statement TEXT NOT NULL,
    
    -- 查询配置
    query_pagination BOOLEAN DEFAULT true,
    query_search_fields TEXT,
    query_sortable_fields TEXT,
    
    -- 创建配置
    create_validation_rules TEXT,
    
    -- 更新配置
    update_updatable_fields TEXT,
    update_validation_rules TEXT,

    -- 其他配置
    other_rules TEXT,
    
    -- 元数据
    description TEXT,
    tags VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    version INTEGER DEFAULT 1,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(connection_id, name)
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_table_configurations_connection ON table_configurations(connection_id);
CREATE INDEX IF NOT EXISTS idx_table_configurations_name ON table_configurations(name);
CREATE INDEX IF NOT EXISTS idx_table_configurations_active ON table_configurations(is_active);

-- 插入示例表配置
INSERT INTO table_configurations (
    connection_id, name, table_name, create_statement,
    query_pagination, query_search_fields, query_sortable_fields,
    create_validation_rules, update_updatable_fields, update_validation_rules,
    description
) VALUES (
    'app_db', 
    'users_config', 
    'users',
    'CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(100) NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, age INTEGER, status VARCHAR(20) DEFAULT ''active'', created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)',
    true,
    '[{"field":"name","type":"fuzzy"},{"field":"email","type":"exact"},{"field":"status","type":"single","dict_source":{"table":"users","field":"status","sort_order":"ASC"}},{"field":"age","type":"range"}]',
    '["id","name","email","age","created_at","updated_at"]',
    '{"name":"required,min=2,max=100","email":"required,email,max=255","age":"min=0,max=150","status":"oneof=active inactive pending"}',
    '["name","email","age","status"]',
    '{"name":"min=2,max=100","email":"email,max=255","age":"min=0,max=150","status":"oneof=active inactive pending"}',
    '用户表CRUD配置'
) ON CONFLICT (connection_id, name) DO NOTHING;

-- 创建更新时间戳触发器函数
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- 创建触发器
DROP TRIGGER IF EXISTS update_table_configurations_updated_at ON table_configurations;
CREATE TRIGGER update_table_configurations_updated_at
    BEFORE UPDATE ON table_configurations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
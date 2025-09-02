-- CRUD Generator MySQL 数据库初始化脚本

-- 表配置管理表
CREATE TABLE IF NOT EXISTS table_configurations (
    id INT AUTO_INCREMENT PRIMARY KEY,
    connection_id VARCHAR(100) NOT NULL,
    name VARCHAR(100) NOT NULL,
    table_name VARCHAR(100) NOT NULL,
    create_statement TEXT NOT NULL,
    
    -- 查询配置
    query_pagination BOOLEAN DEFAULT true,
    query_display_fields TEXT,  -- 展示字段配置
    query_search_fields TEXT,   -- 搜索字段配置
    query_sortable_fields TEXT, -- 可排序字段配置
    
    -- 创建配置
    create_creatable_fields TEXT, -- 可创建字段配置
    create_validation_rules TEXT, -- 创建验证规则
    create_default_values TEXT,   -- 默认值配置
    
    -- 更新配置
    update_updatable_fields TEXT, -- 可更新字段配置
    update_validation_rules TEXT, -- 更新验证规则

    -- 其他配置
    other_rules TEXT, -- 用于存储其他规则或配置
    
    -- 元数据
    description TEXT,
    tags VARCHAR(255),
    is_active BOOLEAN DEFAULT true,
    version INT DEFAULT 1,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_connection_name (connection_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 创建索引
CREATE INDEX idx_table_configurations_connection ON table_configurations(connection_id);
CREATE INDEX idx_table_configurations_name ON table_configurations(name);
CREATE INDEX idx_table_configurations_active ON table_configurations(is_active);

-- 插入示例表配置
INSERT IGNORE INTO table_configurations (
    connection_id, name, table_name, create_statement,
    query_pagination, query_display_fields, query_search_fields, query_sortable_fields,
    create_creatable_fields, create_validation_rules, create_default_values,
    update_updatable_fields, update_validation_rules,
    description
) VALUES (
    'app_db', 
    'users_config', 
    'users',
    'CREATE TABLE users (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(100) NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, age INT, status VARCHAR(20) DEFAULT ''active'', created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)',
    true,
    '[{"field":"id","label":"ID","type":"number"},{"field":"name","label":"姓名","type":"text"},{"field":"email","label":"邮箱","type":"text"},{"field":"age","label":"年龄","type":"number"},{"field":"status","label":"状态","type":"text"}]',
    '[{"field":"name","type":"fuzzy"},{"field":"email","type":"exact"},{"field":"status","type":"single","dict_source":{"table":"users","field":"status","sort_order":"ASC"}},{"field":"age","type":"range"}]',
    '["id","name","email","age","created_at","updated_at"]',
    '[{"field":"name","type":"text","required":true},{"field":"email","type":"email","required":true},{"field":"age","type":"number","required":false},{"field":"status","type":"select","required":false}]',
    '{"name":"required,min=2,max=100","email":"required,email,max=255","age":"min=0,max=150","status":"oneof=active inactive pending"}',
    '[{"field":"status","value":"active"}]',
    '[{"field":"name","type":"text","required":true},{"field":"email","type":"email","required":true},{"field":"age","type":"number","required":false},{"field":"status","type":"select","required":false}]',
    '{"name":"min=2,max=100","email":"email,max=255","age":"min=0,max=150","status":"oneof=active inactive pending"}',
    '用户表CRUD配置'
);
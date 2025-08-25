# CRUD Generator

现代化的Go CRUD API生成器，支持GORM、MySQL和PostgreSQL，提供完整的Web UI管理界面。

## 功能特性

- 🚀 **RESTful API自动生成** - 基于数据库表配置生成完整的CRUD API
- 📊 **智能查询支持** - 分页、排序、多种搜索类型（模糊搜索、精确匹配、范围查询等）
- ✅ **数据验证** - 基于go-playground/validator的强大验证功能
- 📚 **动态字典** - 从数据库动态获取下拉选项和枚举值
- 🗄️ **多数据库支持** - PostgreSQL和MySQL双数据库支持
- 🎯 **灵活配置** - 数据库存储的配置系统，支持运行时修改
- 🖥️ **现代化Web UI** - 完整的管理界面，配置即时生效

## 快速开始

### 1. 安装依赖

```bash
go mod tidy
```

### 2. 启动服务

```bash
# 方式1: 使用启动脚本
./start-webui.sh

# 方式2: 直接编译运行
go build -o crud-generator .
./crud-generator

# 方式3: 直接运行
go run main.go
```

### 3. 访问Web界面

打开浏览器访问 `http://localhost:8080`

## 🌟 核心功能

### RESTful API自动生成

基于表配置自动生成标准的RESTful API端点：

- `GET /api/{config_name}/list` - 列表查询（支持分页、搜索、排序）
- `POST /api/{config_name}/create` - 创建数据
- `PUT /api/{config_name}/update/{id}` - 更新数据
- `DELETE /api/{config_name}/delete/{id}` - 删除数据
- `GET /api/{config_name}/dict/{field}` - 获取字典数据

### 数据库配置管理

- `GET /api/configs` - 获取所有表配置
- `POST /api/configs` - 创建新配置
- `PUT /api/configs/{id}` - 更新配置
- `DELETE /api/configs/{id}` - 删除配置

## 配置示例

### 通过Web UI创建配置

访问 `http://localhost:8080` 使用可视化界面创建表配置。

### API方式创建配置

```bash
curl -X POST http://localhost:8080/api/configs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test",
    "connection_id": 1,
    "table_name": "users",
    "create_statement": "CREATE TABLE users (id SERIAL PRIMARY KEY, name VARCHAR(100) NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, age INTEGER, status VARCHAR(20) DEFAULT '\''active'\'', created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)",
    "query_config": {
      "pagination": true,
      "search_fields": [
        {
          "field": "name",
          "type": "fuzzy"
        },
        {
          "field": "status", 
          "type": "single",
          "dict_source": {
            "table": "users",
            "field": "status",
            "sort_order": "ASC"
          }
        }
      ],
      "sortable_fields": ["id", "name", "created_at"]
    },
    "create_config": {
      "validation_rules": {
        "name": "required,min=2,max=100",
        "email": "required,email"
      }
    },
    "update_config": {
      "updatable_fields": ["name", "email", "age"],
      "validation_rules": {
        "name": "min=2,max=100", 
        "email": "email"
      }
    }
  }'
```

## API使用示例

### 查询数据

```bash
# 基本查询
curl "http://localhost:8080/api/test/list"

# 分页查询
curl "http://localhost:8080/api/test/list?page=1&page_size=10"

# 搜索查询
curl "http://localhost:8080/api/test/list?name=John&status=active"

# 排序查询
curl "http://localhost:8080/api/test/list?sort=created_at&order=desc"
```

### 创建数据

```bash
curl -X POST http://localhost:8080/api/test/create \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Doe",
    "email": "john@example.com", 
    "age": 30
  }'
```

### 更新数据

```bash
curl -X PUT http://localhost:8080/api/test/update/1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "John Smith",
    "age": 31
  }'
```

### 删除数据

```bash
curl -X DELETE http://localhost:8080/api/test/delete/1
```

### 获取字典数据

```bash
curl "http://localhost:8080/api/test/dict/status"
```

## 配置说明

### 搜索类型

- `fuzzy`: 模糊搜索，使用ILIKE
- `exact`: 精确匹配
- `multi`: 多选，使用IN查询
- `single`: 单选
- `range`: 范围查询，支持min/max

### 验证规则

支持所有go-playground/validator的验证标签：

- `required`: 必填
- `min`, `max`: 最小/最大长度或值
- `email`: 邮箱格式
- `oneof`: 枚举值
- 更多规则参见：https://pkg.go.dev/github.com/go-playground/validator/v10

### 字典数据源

```json
{
  "field": "status",
  "type": "single",
  "dict_source": {
    "table": "users",
    "field": "status",
    "sort_order": "ASC",
    "where": "status IS NOT NULL"
  }
}
```

## 数据库配置

系统支持配置多个数据库连接，配置信息存储在主数据库中。

### 支持的数据库类型

- **PostgreSQL** - 推荐使用，功能最完整
- **MySQL** - 完全支持，兼容性良好

### 连接配置示例

数据库连接配置通过配置文件 `configs/db.json` 管理：

```json
{
  "connections": [
    {
      "id": 1,
      "name": "PostgreSQL主库",
      "type": "postgres",
      "host": "localhost",
      "port": 5432,
      "database": "testdb",
      "username": "postgres",
      "password": "password"
    },
    {
      "id": 2, 
      "name": "MySQL从库",
      "type": "mysql",
      "host": "localhost",
      "port": 3306,
      "database": "testdb",
      "username": "root",
      "password": "password"
    }
  ]
}
```

## 运行示例

1. 启动服务：
```bash
./start-webui.sh
```

2. 访问Web界面：
```bash
open http://localhost:8080
```

3. 运行测试：
```bash
go test ./tests/...
```

## 架构设计

```
crud-generator/
├── main.go             # 主服务入口，HTTP API服务器
├── builder/            # 核心CRUD构建器
├── config/             # 配置加载和管理
├── database/           # 数据库连接管理器  
├── generator/          # SQL生成器和查询构建
├── models/             # 数据模型定义
├── parser/             # 数据库结构解析器
├── services/           # 业务逻辑服务层
├── types/              # 类型定义和接口
├── validator/          # 数据验证器
├── webui/              # Web前端界面
├── cmd/                # 命令行工具
├── configs/            # 配置文件目录
├── examples/           # 示例代码
├── sql/                # 数据库初始化脚本
└── tests/              # 测试用例
```

### 核心组件

- **API Server** (`main.go`) - RESTful API服务，处理HTTP请求
- **配置服务** (`services/`) - 管理表配置的CRUD操作
- **CRUD服务** (`services/`) - 动态生成数据库操作的业务逻辑
- **数据库管理器** (`database/`) - 多数据库连接管理
- **查询构建器** (`generator/`) - 动态SQL生成和执行
- **Web UI** (`webui/`) - 现代化的管理界面

## 环境变量

- `CRUD_DB_PATH`: 主数据库文件路径，默认为 `./main.db`
- `CRUD_CONFIG_PATH`: 配置文件目录，默认为 `./configs`

## 许可证

MIT License
# CRUD Generator - Go 包

一个强大且灵活的 Go 应用程序 CRUD（创建、读取、更新、删除）生成器，提供编程 API 和可嵌入的 Web UI 来进行数据库操作。

## 特性

- 🚀 **轻松集成**：只需几行代码即可为您的 Go 应用添加 CRUD 功能
- 🎨 **可嵌入的 Web UI**：美观且响应式的 Web 界面，可嵌入到您的应用程序中
- 🔧 **编程 API**：对 CRUD 操作的完全编程控制
- 🗄️ **多数据库支持**：支持 PostgreSQL、MySQL，具备连接池功能
- 🛡️ **认证就绪**：内置 JWT 认证（可选）
- ⚡ **高性能**：基于 GORM，为生产环境优化
- 🎛️ **可配置**：灵活的表、字段和操作配置选项

## 安装

```bash
go get github.com/otkinlife/crud-generator
```

## 快速开始

### 使用现有数据库连接的基本用法

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/gin-gonic/gin"
    crudgen "github.com/otkinlife/crud-generator"
    "gorm.io/gorm"
)

func main() {
    // 假设您有一个现有的 GORM 数据库连接
    var db *gorm.DB // 您现有的数据库连接
    
    // 使用您的数据库创建 CRUD 生成器
    config := crudgen.DefaultConfig()
    config.UIBasePath = "/admin"   // 在 /admin 提供 UI
    config.APIBasePath = "/api/v1" // API 在 /api/v1
    
    generator, err := crudgen.NewWithGormDB(db, "main", config)
    if err != nil {
        log.Fatal("创建 CRUD 生成器失败：", err)
    }
    defer generator.Close()
    
    // 创建 Gin 路由器
    router := gin.Default()
    
    // 注册 CRUD 路由
    generator.RegisterRoutes(router)
    
    // 您的其他路由
    router.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "你好世界"})
    })
    
    // 启动服务器
    log.Println("服务器在 :8080 启动")
    log.Printf("CRUD UI 可访问地址：http://localhost:8080%s", config.UIBasePath)
    log.Fatal(http.ListenAndServe(":8080", router))
}
```

### 完整配置示例

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/gin-gonic/gin"
    crudgen "github.com/otkinlife/crud-generator"
)

func main() {
    // 创建配置
    config := &crudgen.Config{
        EnableAuth:       true,
        JWTSecret:       "your-secret-key",
        TokenExpireHours: 24,
        UIEnabled:       true,
        UIBasePath:      "/admin",
        APIBasePath:     "/api/v1",
        DatabaseConfig: map[string]crudgen.DatabaseConnection{
            "main": {
                Type:         "postgresql",
                Host:         "localhost",
                Port:         5432,
                Database:     "myapp",
                Username:     "postgres",
                Password:     "password",
                SSLMode:      "disable",
                MaxIdleConns: 10,
                MaxOpenConns: 100,
            },
            "analytics": {
                Type:         "mysql",
                Host:         "localhost",
                Port:         3306,
                Database:     "analytics",
                Username:     "root",
                Password:     "password",
                MaxIdleConns: 5,
                MaxOpenConns: 50,
            },
        },
    }
    
    // 创建 CRUD 生成器
    generator, err := crudgen.New(config)
    if err != nil {
        log.Fatal("创建 CRUD 生成器失败：", err)
    }
    defer generator.Close()
    
    // 通过编程方式添加表配置
    userTableConfig := &crudgen.TableConfig{
        Name:         "users",
        TableName:    "users",
        ConnectionID: "main",
        CreateStatement: `
            CREATE TABLE users (
                id SERIAL PRIMARY KEY,
                username VARCHAR(50) UNIQUE NOT NULL,
                email VARCHAR(100) UNIQUE NOT NULL,
                full_name VARCHAR(100),
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        `,
        QueryPagination: true,
        Description: "用户管理表",
        IsActive: true,
        Version: 1,
    }
    
    if err := generator.AddTableConfig(userTableConfig); err != nil {
        log.Printf("警告：添加用户表配置失败：%v", err)
    }
    
    // 创建路由器并注册路由
    router := gin.Default()
    generator.RegisterRoutes(router)
    
    // 启动服务器
    log.Println("服务器在 :8080 启动")
    log.Printf("CRUD 管理 UI：http://localhost:8080%s", config.UIBasePath)
    log.Printf("API 端点：http://localhost:8080%s", config.APIBasePath)
    log.Fatal(http.ListenAndServe(":8080", router))
}
```

### 编程式 CRUD 操作

```go
package main

import (
    "fmt"
    "log"
    
    crudgen "github.com/otkinlife/crud-generator"
)

func main() {
    // 设置生成器（同上）
    generator, err := crudgen.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer generator.Close()
    
    // 创建记录
    userData := map[string]interface{}{
        "username":  "johndoe",
        "email":     "john@example.com",
        "full_name": "John Doe",
    }
    
    createResult, err := generator.Create("users", userData)
    if err != nil {
        log.Printf("创建失败：%v", err)
    } else if createResult.Success { 
        fmt.Printf("用户已创建：%+v\n", createResult.Data)
    }
    
    // 带分页和搜索的列表记录
    listParams := &crudgen.QueryParams{
        Page:     1,
        PageSize: 10,
        Search: map[string]interface{}{
            "username": "john", // 搜索包含 "john" 的用户名
        },
        Sort: []crudgen.SortField{
            {Field: "created_at", Order: crudgen.SortOrderDESC},
        },
    }
    
    listResult, err := generator.List("users", listParams)
    if err != nil {
        log.Printf("列表查询失败：%v", err)
    } else {
        fmt.Printf("找到 %d 个用户（第 %d/%d 页）：\n", 
            listResult.Total, listResult.Page, listResult.TotalPages)
        for _, user := range listResult.Data {
            fmt.Printf("  - %s (%s)\n", user["username"], user["email"])
        }
    }
    
    // 更新记录
    updateData := map[string]interface{}{
        "full_name": "John Smith",
    }
    
    updateResult, err := generator.Update("users", 1, updateData)
    if err != nil {
        log.Printf("更新失败：%v", err)
    } else if updateResult.Success {
        fmt.Println("用户更新成功")
    }
    
    // 删除记录
    deleteResult, err := generator.Delete("users", 1)
    if err != nil {
        log.Printf("删除失败：%v", err)
    } else if deleteResult.Success {
        fmt.Println("用户删除成功")
    }
}
```

### 在现有应用程序中嵌入 UI

```go
package main

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    crudgen "github.com/otkinlife/crud-generator"
)

func main() {
    // 您现有的 Gin 应用程序
    router := gin.Default()
    
    // 您现有的路由
    router.GET("/", homePage)
    router.GET("/dashboard", dashboard)
    
    // 创建 CRUD 生成器
    config := crudgen.DefaultConfig()
    config.UIBasePath = "/admin/crud"  // 嵌入到 /admin/crud
    config.APIBasePath = "/api/admin"  // API 在 /api/admin
    config.EnableAuth = false          // 如果您在其他地方处理认证，则禁用认证
    
    generator, err := crudgen.NewWithGormDB(yourDB, "main", config)
    if err != nil {
        panic(err)
    }
    defer generator.Close()
    
    // 仅在管理员部分注册
    adminGroup := router.Group("/admin")
    {
        adminGroup.Use(yourAuthMiddleware()) // 您现有的认证
        
        // 在 /admin 下注册 CRUD 路由
        generator.RegisterRoutes(adminGroup)
    }
    
    http.ListenAndServe(":8080", router)
}
```

### 仅 API 模式（无 UI）

```go
package main

import (
    "net/http"
    
    "github.com/gin-gonic/gin" 
    crudgen "github.com/otkinlife/crud-generator"
)

func main() {
    config := crudgen.DefaultConfig()
    config.UIEnabled = false // 禁用 UI
    config.APIBasePath = "/api/v1"
    
    generator, err := crudgen.New(config)
    if err != nil {
        panic(err)
    }
    defer generator.Close()
    
    router := gin.Default()
    
    // 仅注册 API 路由
    generator.RegisterAPIRoutes(router)
    
    http.ListenAndServe(":8080", router)
}
```

### 微服务架构

```go
// crud-service/main.go
package main

import (
    "net/http"
    
    crudgen "github.com/otkinlife/crud-generator"
)

func main() {
    config := &crudgen.Config{
        UIEnabled:    true,
        UIBasePath:   "/ui",
        APIBasePath:  "/api",
        DatabaseConfig: map[string]crudgen.DatabaseConnection{
            "main": {
                Type:     "postgresql",
                Host:     "postgres-service",
                Port:     5432,
                Database: "app_data",
                Username: "postgres",
                Password: "password",
            },
        },
    }
    
    generator, err := crudgen.New(config)
    if err != nil {
        panic(err)
    }
    defer generator.Close()
    
    // 获取整个 CRUD 功能的 HTTP 处理器
    handler := generator.GetFullHandler()
    
    // 作为独立服务运行
    http.ListenAndServe(":8080", handler)
}
```

## API 端点

当您集成 CRUD 生成器时，它提供以下 API 端点：

### 配置管理

- `GET /api/configs` - 列出所有表配置
- `POST /api/configs` - 创建新的表配置
- `GET /api/configs/:id` - 通过 ID 获取配置
- `GET /api/configs/by-name/:name` - 通过名称获取配置
- `PUT /api/configs/:id` - 更新配置
- `DELETE /api/configs/:id` - 删除配置

### 数据库操作

- `GET /api/connections` - 列出数据库连接
- `POST /api/connections/:id/test` - 测试数据库连接

### CRUD 操作（每个配置的表）

- `GET /api/:table_name/list` - 列出记录（支持分页、搜索、排序）
- `POST /api/:table_name/create` - 创建新记录
- `PUT /api/:table_name/update/:id` - 更新现有记录
- `DELETE /api/:table_name/delete/:id` - 删除记录
- `GET /api/:table_name/dict/:field` - 获取字段的字典值

### 认证（如果启用）

- `POST /api/auth/login` - 用户登录
- `POST /api/auth/refresh` - 刷新 JWT 令牌

## 配置选项

### 数据库连接

```go
type DatabaseConnection struct {
    Type         string `json:"type"`          // "postgresql" 或 "mysql"
    Host         string `json:"host"`
    Port         int    `json:"port"`
    Database     string `json:"database"`
    Username     string `json:"username"`
    Password     string `json:"password"`
    SSLMode      string `json:"ssl_mode"`      // 仅 PostgreSQL
    MaxIdleConns int    `json:"max_idle_conns"`
    MaxOpenConns int    `json:"max_open_conns"`
}
```

### 表配置

```go
type TableConfig struct {
    Name         string `json:"name"`           // 配置名称
    TableName    string `json:"table_name"`     // 实际数据库表名
    ConnectionID string `json:"connection_id"`  // 使用的数据库连接
    
    CreateStatement string `json:"create_statement"` // SQL CREATE TABLE 语句
    
    // UI 配置（JSON 字符串）
    QueryDisplayFields  string `json:"query_display_fields"`  // 列表中显示的字段
    QuerySearchFields   string `json:"query_search_fields"`   // 可搜索的字段
    QuerySortableFields string `json:"query_sortable_fields"` // 可排序的字段
    CreateCreatableFields string `json:"create_creatable_fields"` // 创建表单字段
    UpdateUpdatableFields string `json:"update_updatable_fields"` // 编辑表单字段
    
    Description string `json:"description"`
    Tags        string `json:"tags"`
    IsActive    bool   `json:"is_active"`
    Version     int    `json:"version"`
}
```

## 高级功能

### 自定义字段类型

系统支持各种表单字段类型：

- `text` - 文本输入
- `textarea` - 多行文本
- `number` - 数字输入
- `date` - 日期选择器
- `datetime` - 日期时间选择器
- `select` - 下拉选择
- `checkbox` - 复选框输入

### 搜索类型

配置不同的搜索行为：

- `fuzzy` - 模糊搜索（LIKE）
- `exact` - 精确匹配
- `range` - 数字范围
- `single` - 单选下拉框
- `multi_select` - 多选
- `date_range` - 日期范围选择器

### 验证

添加验证规则以确保数据完整性：

```go
// 字段验证配置
type FieldValidation struct {
    MinLength    *int   `json:"min_length,omitempty"`    // 最小长度
    MaxLength    *int   `json:"max_length,omitempty"`    // 最大长度
    Min          *int   `json:"min,omitempty"`           // 最小值
    Max          *int   `json:"max,omitempty"`           // 最大值
    Pattern      string `json:"pattern,omitempty"`       // 正则表达式
    ErrorMessage string `json:"error_message,omitempty"` // 自定义错误消息
}
```

## 项目结构

```
crud-generator/
├── crudgen.go              # 主要包接口 (package crudgen)
├── types.go                # 外部 API 类型定义 (package crudgen)
├── handlers.go             # HTTP 处理器 (package crudgen)
├── service_adapters.go     # 服务适配器 (package crudgen)
├── database_manager.go     # 数据库管理器 (package crudgen)
├── cmd/
│   └── crud-generator/     # 独立应用程序
│       └── main.go         # 可执行程序 (package main)
├── examples/               # 使用示例
│   ├── main.go             # 基本示例 (package main)
│   └── package_usage/      # 包使用示例
│       └── main.go         # 包使用示例 (package main)
├── services/               # 内部服务实现
├── types/                  # 内部类型定义
├── models/                 # 数据模型
├── validator/              # 验证器
├── webui/                  # Web UI 静态文件
└── ...                     # 其他支持目录
```

## 使用方式

1. **作为库包使用**：
   ```go
   import crudgen "github.com/otkinlife/crud-generator"
   ```

2. **作为独立应用运行**：
   ```bash
   go run cmd/crud-generator/main.go
   # 或者
   ./start-webui.sh
   ```

## 贡献

我们欢迎贡献！请查看 [CONTRIBUTING.md](CONTRIBUTING.md) 了解详情。

## 许可证

本项目采用 MIT 许可证 - 详情请查看 [LICENSE](LICENSE) 文件。

## 支持

- GitHub Issues: [https://github.com/otkinlife/crud-generator/issues](https://github.com/otkinlife/crud-generator/issues)
- 文档: [https://crud-generator.docs.com](https://crud-generator.docs.com)
- 示例: [https://github.com/otkinlife/crud-generator-examples](https://github.com/otkinlife/crud-generator-examples)
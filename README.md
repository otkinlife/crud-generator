# CRUD Generator

现代化的Go CRUD包，可嵌入到现有Go应用中，支持PostgreSQL和MySQL，提供完整的Web UI管理界面。

## 安装

```bash
go get github.com/otkinlife/crud-generator
```

## 使用方法

### 基本用法

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
        UIEnabled:   true,
        UIBasePath:  "/admin",      // Web UI路径
        APIBasePath: "/api/v1",     // API路径前缀
        DatabaseConfig: map[string]crudgen.DatabaseConnection{
            "main": {
                Type:     "postgresql",
                Host:     "localhost",
                Port:     5432,
                Database: "your_db",
                Username: "postgres",
                Password: "password",
            },
        },
    }
    
    // 创建CRUD生成器
    generator, err := crudgen.New(config)
    if err != nil {
        log.Fatal("Failed to create CRUD generator:", err)
    }
    defer generator.Close()
    
    // 创建Gin路由器
    router := gin.Default()
    
    // 注册CRUD路由
    generator.RegisterRoutes(router)
    
    // 启动服务器
    log.Fatal(http.ListenAndServe(":8080", router))
}
```

访问 `http://localhost:8080/admin` 查看Web界面

### 使用现有数据库连接

```go
// 假设你有现有的GORM数据库连接
var db *gorm.DB // 你的现有数据库连接

config := crudgen.DefaultConfig()
config.UIBasePath = "/admin"
config.APIBasePath = "/api/v1"

generator, err := crudgen.NewWithGormDB(db, "main", config)
if err != nil {
    log.Fatal("Failed to create CRUD generator:", err)
}
defer generator.Close()

router := gin.Default()
generator.RegisterRoutes(router)

log.Fatal(http.ListenAndServe(":8080", router))
```

### 集成到现有应用

```go
// 在现有Gin应用中集成
router := gin.Default()

// 你的现有路由
router.GET("/", homePage)
router.GET("/dashboard", dashboard)

// 创建CRUD生成器
config := crudgen.DefaultConfig()
config.UIBasePath = "/admin/crud"  // 嵌入到 /admin/crud
config.APIBasePath = "/api/admin"  // API在 /api/admin

generator, err := crudgen.NewWithGormDB(yourDB, "main", config)
if err != nil {
    panic(err)
}
defer generator.Close()

// 在管理员部分注册
adminGroup := router.Group("/admin")
{
    adminGroup.Use(yourAuthMiddleware()) // 你的认证中间件
    generator.RegisterRoutes(adminGroup) // 会在 /admin/crud 下提供UI
}

http.ListenAndServe(":8080", router)
```

### 仅API模式（无UI）

```go
config := crudgen.DefaultConfig()
config.UIEnabled = false // 禁用UI
config.APIBasePath = "/api/v1"

generator, err := crudgen.New(config)
if err != nil {
    panic(err)
}
defer generator.Close()

router := gin.Default()
generator.RegisterAPIRoutes(router) // 仅注册API路由

http.ListenAndServe(":8080", router)
```

## 配置选项

### 数据库配置

```go
config := &crudgen.Config{
    UIEnabled:   true,           // 启用Web UI，默认: false
    UIBasePath:  "/admin",       // UI路径前缀，默认: "/crud-ui"
    APIBasePath: "/api/v1",      // API路径前缀，默认: "/api"
    
    DatabaseConfig: map[string]crudgen.DatabaseConnection{
        "main": {
            Type:         "postgresql",  // postgresql 或 mysql
            Host:         "localhost",
            Port:         5432,
            Database:     "your_db",
            Username:     "postgres",
            Password:     "password",
            SSLMode:      "disable",     // PostgreSQL SSL模式
            MaxIdleConns: 10,            // 最大空闲连接数
            MaxOpenConns: 100,           // 最大打开连接数
        },
    },
}
```

### 表配置示例

```go
// 添加表配置
userTable := &crudgen.TableConfig{
    Name:         "users",
    TableName:    "users", 
    ConnectionID: "main",
    CreateStatement: `CREATE TABLE users (
        id SERIAL PRIMARY KEY,
        username VARCHAR(50) UNIQUE NOT NULL,
        email VARCHAR(100) UNIQUE NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )`,
    QueryPagination: true,
    Description:     "用户管理表",
    IsActive:        true,
}

err := generator.AddTableConfig(userTable)
if err != nil {
    log.Printf("Failed to add table config: %v", err)
}
```

## 示例

参考 `examples/package_usage/main.go`：

```bash
go run ./examples/package_usage/
```

## 许可证

MIT License
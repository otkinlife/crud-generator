# CRUD Generator - Go Package

A powerful and flexible CRUD (Create, Read, Update, Delete) generator for Go applications that provides both programmatic API and embeddable web UI for database operations.

## Features

- 🚀 **Easy Integration**: Add CRUD functionality to your Go app with just a few lines of code
- 🎨 **Embeddable Web UI**: Beautiful and responsive web interface that can be embedded in your application
- 🔧 **Programmatic API**: Full programmatic control over CRUD operations
- 🗄️ **Multi-Database Support**: PostgreSQL, MySQL support with connection pooling
- 🛡️ **Authentication Ready**: Built-in JWT authentication (optional)
- ⚡ **High Performance**: Optimized for production use with GORM
- 🎛️ **Configurable**: Flexible configuration options for tables, fields, and operations

## Installation

```bash
go get github.com/your-org/crud-generator
```

## Quick Start

### Basic Usage with Existing Database Connection

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/your-org/crud-generator"
    "gorm.io/gorm"
)

func main() {
    // Assume you have an existing GORM database connection
    var db *gorm.DB // Your existing database connection
    
    // Create CRUD generator with your database
    config := crudgen.DefaultConfig()
    config.UIBasePath = "/admin"  // Serve UI at /admin
    config.APIBasePath = "/api/v1"  // API at /api/v1
    
    generator, err := crudgen.NewWithGormDB(db, "main", config)
    if err != nil {
        log.Fatal("Failed to create CRUD generator:", err)
    }
    defer generator.Close()
    
    // Create Gin router
    router := gin.Default()
    
    // Register CRUD routes
    generator.RegisterRoutes(router)
    
    // Your other routes
    router.GET("/", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "Hello World"})
    })
    
    // Start server
    log.Println("Server starting on :8080")
    log.Printf("CRUD UI available at: http://localhost:8080%s", config.UIBasePath)
    log.Fatal(http.ListenAndServe(":8080", router))
}
```

### Full Configuration Example

```go
package main

import (
    "log"
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/your-org/crud-generator"
)

func main() {
    // Create configuration
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
    
    // Create CRUD generator
    generator, err := crudgen.New(config)
    if err != nil {
        log.Fatal("Failed to create CRUD generator:", err)
    }
    defer generator.Close()
    
    // Add table configurations programmatically
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
        Description: "User management table",
        IsActive: true,
        Version: 1,
    }
    
    if err := generator.AddTableConfig(userTableConfig); err != nil {
        log.Printf("Warning: Failed to add user table config: %v", err)
    }
    
    // Create router and register routes
    router := gin.Default()
    generator.RegisterRoutes(router)
    
    // Start server
    log.Println("Server starting on :8080")
    log.Printf("CRUD Admin UI: http://localhost:8080%s", config.UIBasePath)
    log.Printf("API Endpoints: http://localhost:8080%s", config.APIBasePath)
    log.Fatal(http.ListenAndServe(":8080", router))
}
```

### Programmatic CRUD Operations

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/your-org/crud-generator"
)

func main() {
    // Setup generator (same as above)
    generator, err := crudgen.New(config)
    if err != nil {
        log.Fatal(err)
    }
    defer generator.Close()
    
    // Create a record
    userData := map[string]interface{}{
        "username":  "johndoe",
        "email":     "john@example.com",
        "full_name": "John Doe",
    }
    
    createResult, err := generator.Create("users", userData)
    if err != nil {
        log.Printf("Create failed: %v", err)
    } else if createResult.Success {
        fmt.Printf("User created: %+v\n", createResult.Data)
    }
    
    // List records with pagination and search
    listParams := &crudgen.QueryParams{
        Page:     1,
        PageSize: 10,
        Search: map[string]interface{}{
            "username": "john",  // Search for usernames containing "john"
        },
        Sort: []crudgen.SortField{
            {Field: "created_at", Order: crudgen.SortOrderDESC},
        },
    }
    
    listResult, err := generator.List("users", listParams)
    if err != nil {
        log.Printf("List failed: %v", err)
    } else {
        fmt.Printf("Found %d users (page %d/%d):\n", 
            listResult.Total, listResult.Page, listResult.TotalPages)
        for _, user := range listResult.Data {
            fmt.Printf("  - %s (%s)\n", user["username"], user["email"])
        }
    }
    
    // Update a record
    updateData := map[string]interface{}{
        "full_name": "John Smith",
    }
    
    updateResult, err := generator.Update("users", 1, updateData)
    if err != nil {
        log.Printf("Update failed: %v", err)
    } else if updateResult.Success {
        fmt.Println("User updated successfully")
    }
    
    // Delete a record
    deleteResult, err := generator.Delete("users", 1)
    if err != nil {
        log.Printf("Delete failed: %v", err)
    } else if deleteResult.Success {
        fmt.Println("User deleted successfully")
    }
}
```

### Embedding UI in Existing Application

```go
package main

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/your-org/crud-generator"
)

func main() {
    // Your existing Gin application
    router := gin.Default()
    
    // Your existing routes
    router.GET("/", homePage)
    router.GET("/dashboard", dashboard)
    
    // Create CRUD generator
    config := crudgen.DefaultConfig()
    config.UIBasePath = "/admin/crud"  // Embed at /admin/crud
    config.APIBasePath = "/api/admin"   // API at /api/admin
    config.EnableAuth = false           // Disable auth if you handle it elsewhere
    
    generator, err := crudgen.NewWithGormDB(yourDB, "main", config)
    if err != nil {
        panic(err)
    }
    defer generator.Close()
    
    // Register only in admin section
    adminGroup := router.Group("/admin")
    {
        adminGroup.Use(yourAuthMiddleware()) // Your existing auth
        
        // Register CRUD routes under /admin
        generator.RegisterRoutes(adminGroup)
    }
    
    http.ListenAndServe(":8080", router)
}
```

### API-Only Mode (No UI)

```go
package main

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
    "github.com/your-org/crud-generator"
)

func main() {
    config := crudgen.DefaultConfig()
    config.UIEnabled = false  // Disable UI
    config.APIBasePath = "/api/v1"
    
    generator, err := crudgen.New(config)
    if err != nil {
        panic(err)
    }
    defer generator.Close()
    
    router := gin.Default()
    
    // Register only API routes
    generator.RegisterAPIRoutes(router)
    
    http.ListenAndServe(":8080", router)
}
```

### Microservice Architecture

```go
// crud-service/main.go
package main

import (
    "net/http"
    
    "github.com/your-org/crud-generator"
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
    
    // Get HTTP handler for the entire CRUD functionality
    handler := generator.GetFullHandler()
    
    // Run as standalone service
    http.ListenAndServe(":8080", handler)
}
```

## API Endpoints

When you integrate the CRUD generator, it provides the following API endpoints:

### Configuration Management

- `GET /api/configs` - List all table configurations
- `POST /api/configs` - Create new table configuration  
- `GET /api/configs/:id` - Get configuration by ID
- `GET /api/configs/by-name/:name` - Get configuration by name
- `PUT /api/configs/:id` - Update configuration
- `DELETE /api/configs/:id` - Delete configuration

### Database Operations

- `GET /api/connections` - List database connections
- `POST /api/connections/:id/test` - Test database connection

### CRUD Operations (per configured table)

- `GET /api/:table_name/list` - List records with pagination, search, sorting
- `POST /api/:table_name/create` - Create new record
- `PUT /api/:table_name/update/:id` - Update existing record
- `DELETE /api/:table_name/delete/:id` - Delete record
- `GET /api/:table_name/dict/:field` - Get dictionary values for field

### Authentication (if enabled)

- `POST /api/auth/login` - User login
- `POST /api/auth/refresh` - Refresh JWT token

## Configuration Options

### Database Connection

```go
type DatabaseConnection struct {
    Type         string `json:"type"`          // "postgresql" or "mysql"
    Host         string `json:"host"`
    Port         int    `json:"port"`
    Database     string `json:"database"`
    Username     string `json:"username"`
    Password     string `json:"password"`
    SSLMode      string `json:"ssl_mode"`      // PostgreSQL only
    MaxIdleConns int    `json:"max_idle_conns"`
    MaxOpenConns int    `json:"max_open_conns"`
}
```

### Table Configuration

```go
type TableConfig struct {
    Name         string `json:"name"`           // Configuration name
    TableName    string `json:"table_name"`     // Actual database table name
    ConnectionID string `json:"connection_id"`  // Database connection to use
    
    CreateStatement string `json:"create_statement"` // SQL CREATE TABLE statement
    
    // UI Configuration (JSON strings)
    QueryDisplayFields  string `json:"query_display_fields"`  // Fields to show in list
    QuerySearchFields   string `json:"query_search_fields"`   // Searchable fields
    QuerySortableFields string `json:"query_sortable_fields"` // Sortable fields
    CreateCreatableFields string `json:"create_creatable_fields"` // Fields for create form
    UpdateUpdatableFields string `json:"update_updatable_fields"` // Fields for edit form
    
    Description string `json:"description"`
    Tags        string `json:"tags"`
    IsActive    bool   `json:"is_active"`
    Version     int    `json:"version"`
}
```

## Advanced Features

### Custom Field Types

The system supports various field types for forms:

- `text` - Text input
- `textarea` - Multi-line text
- `number` - Numeric input
- `date` - Date picker
- `datetime` - Date and time picker
- `select` - Dropdown selection
- `checkbox` - Checkbox input

### Search Types

Configure different search behaviors:

- `fuzzy` - LIKE search
- `exact` - Exact match
- `range` - Numeric range
- `single` - Single select dropdown
- `multi_select` - Multiple selection
- `date_range` - Date range picker

### Validation

Add validation rules to ensure data integrity:

```go
validationRules := map[string][]string{
    "email": {"required", "email"},
    "age":   {"required", "min:0", "max:150"},
    "username": {"required", "min:3", "max:50", "unique"},
}
```

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- GitHub Issues: [https://github.com/your-org/crud-generator/issues](https://github.com/your-org/crud-generator/issues)
- Documentation: [https://crud-generator.docs.com](https://crud-generator.docs.com)
- Examples: [https://github.com/your-org/crud-generator-examples](https://github.com/your-org/crud-generator-examples)
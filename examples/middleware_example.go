package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	crudgen "github.com/otkinlife/crud-generator"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 示例：自定义鉴权中间件
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取token
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Missing authorization token",
			})
			c.Abort()
			return
		}

		// 验证token (这里是示例逻辑)
		if !strings.HasPrefix(token, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid token format",
			})
			c.Abort()
			return
		}

		// 解析token并验证 (示例)
		actualToken := strings.TrimPrefix(token, "Bearer ")
		if actualToken != "valid-token-123" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid token",
			})
			c.Abort()
			return
		}

		// 设置用户信息到context
		c.Set("user_id", "user123")
		c.Set("username", "admin")
		c.Set("role", "admin")

		c.Next()
	}
}

// 示例：IP限制中间件
func ipLimitMiddleware() gin.HandlerFunc {
	allowedIPs := []string{"127.0.0.1", "::1"} // 允许的IP列表

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		allowed := false
		for _, ip := range allowedIPs {
			if clientIP == ip {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   fmt.Sprintf("IP %s is not allowed", clientIP),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// 示例：签名验证中间件
func signatureMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		signature := c.GetHeader("X-Signature")
		timestamp := c.GetHeader("X-Timestamp")

		if signature == "" || timestamp == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Missing signature or timestamp",
			})
			c.Abort()
			return
		}

		// 这里可以添加签名验证逻辑
		// 例如：验证HMAC签名等

		c.Next()
	}
}

// 示例：角色权限中间件
func roleMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "User role not found",
			})
			c.Abort()
			return
		}

		userRole, ok := role.(string)
		if !ok || userRole != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Required role: %s", requiredRole),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func main() {
	// 创建数据库连接
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 配置中间件
	middlewareConfig := crudgen.NewMiddlewareBuilder().
		// 全局中间件：应用于所有路由
		Global(
			gin.Recovery(), // 恢复中间件
			gin.Logger(),   // 日志中间件
		).
		// API中间件：仅应用于API路由
		API(
			authMiddleware(),    // 鉴权中间件
			ipLimitMiddleware(), // IP限制中间件
		).
		// UI中间件：仅应用于UI路由
		UI(
		// UI可能需要不同的鉴权策略
		).
		// 特定路由中间件
		Route("/configs",
			roleMiddleware("admin"), // 配置管理需要管理员权限
			signatureMiddleware(),   // 签名验证
		).
		Route("/crud").// CRUD操作可能需要不同的权限

		// 公开路由：跳过全局中间件中的鉴权
		Public(
			"/api/connections", // 连接信息可以公开
			"/health",          // 健康检查
			"/auth/login",      // 登录接口
		).
		Build()

	// 创建CRUD generator配置
	config := &crudgen.Config{
		UIEnabled:        true,
		UIBasePath:       "/admin",
		APIBasePath:      "/api",
		MiddlewareConfig: middlewareConfig,
	}

	// 创建CRUD generator
	generator, err := crudgen.NewWithGormDB(db, "main", config)
	if err != nil {
		log.Fatal("Failed to create CRUD generator:", err)
	}

	// 创建Gin路由器
	router := gin.New()

	// 添加自定义路由（如登录接口）
	router.POST("/auth/login", func(c *gin.Context) {
		var loginReq struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := c.ShouldBindJSON(&loginReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid request format",
			})
			return
		}

		// 验证用户名密码 (示例)
		if loginReq.Username == "admin" && loginReq.Password == "admin123" {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"token": "valid-token-123",
					"user": gin.H{
						"id":       "user123",
						"username": "admin",
						"role":     "admin",
					},
				},
			})
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid credentials",
			})
		}
	})

	// 健康检查接口
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"status":  "healthy",
		})
	})

	// 注册CRUD generator路由
	generator.RegisterRoutes(router)

	fmt.Println("Server starting on :8080")
	fmt.Println("API endpoints:")
	fmt.Println("  POST /auth/login - Login with username=admin, password=admin123")
	fmt.Println("  GET  /health - Health check")
	fmt.Println("  GET  /api/connections - List connections (public)")
	fmt.Println("  GET  /api/configs - List configs (requires auth + admin role)")
	fmt.Println("  GET  /admin - Web UI (requires auth)")

	log.Fatal(router.Run(":8080"))
}

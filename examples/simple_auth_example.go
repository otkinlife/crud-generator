package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	crudgen "github.com/otkinlife/crud-generator"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 简单的JWT鉴权中间件示例
func simpleAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")

		// 简单的token验证
		if token != "Bearer secret-token" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or missing token",
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

	// 配置中间件 - 仅对API路由应用鉴权
	middlewareConfig := crudgen.NewMiddlewareBuilder().
		API(simpleAuthMiddleware()). // 所有API都需要鉴权
		Public("/api/connections").  // 连接信息公开访问
		Build()

	// 创建配置
	config := &crudgen.Config{
		UIEnabled:        true,
		UIBasePath:       "/crud-ui",
		APIBasePath:      "/api",
		MiddlewareConfig: middlewareConfig,
	}

	// 创建CRUD generator
	generator, err := crudgen.NewWithGormDB(db, "main", config)
	if err != nil {
		log.Fatal("Failed to create CRUD generator:", err)
	}

	// 创建路由器并注册路由
	router := gin.Default()
	generator.RegisterRoutes(router)

	log.Println("Server starting on :8080")
	log.Println("Try:")
	log.Println("  GET  /api/connections (no auth required)")
	log.Println("  GET  /api/configs (requires: Authorization: Bearer secret-token)")

	router.Run(":8080")
}

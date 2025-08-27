package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/otkinlife/go_tools/jwt_tools"
	"github.com/otkinlife/go_tools/logger_tools"
)

const (
	// JWT配置常量
	JWTSecretKey = "crud-generator-secret-key-2024"
	TokenExpire  = time.Hour * 2 // 2小时过期
)

var (
	// 错误定义
	ErrInvalidCredentials = errors.New("invalid username or password")
)

// AuthResponse 认证响应结构
type AuthResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// TokenResponse token响应结构
type TokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	UserInfo  UserInfo  `json:"user_info"`
}

// UserInfo 用户信息结构
type UserInfo struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// LoginRequest 登录请求结构
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// JWTAuth JWT认证中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建日志上下文
		ctx := logger_tools.NewContext(c.Request.Context())
		ctx = logger_tools.WithFields(ctx, map[string]any{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"ip":     c.ClientIP(),
		})

		// 从header获取token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			logger_tools.Warn(ctx, "Missing Authorization header")
			c.JSON(http.StatusUnauthorized, AuthResponse{
				Success: false,
				Error:   "Missing authorization header",
			})
			c.Abort()
			return
		}

		// 检查Bearer格式
		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			logger_tools.Warn(ctx, "Invalid authorization format")
			c.JSON(http.StatusUnauthorized, AuthResponse{
				Success: false,
				Error:   "Invalid authorization format",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]

		// 验证token
		config := jwt_tools.JwtConfig{
			SecretKey:     JWTSecretKey,
			SigningMethod: jwt.SigningMethodHS256,
			ExpireTime:    TokenExpire,
		}

		tokenBuilder := jwt_tools.NewTokenBuilder(config)
		tokenBuilder.SetToken(token)

		if err := tokenBuilder.VerifyToken(); err != nil {
			logger_tools.Warn(ctx, "Token verification failed:", err.Error())
			c.JSON(http.StatusUnauthorized, AuthResponse{
				Success: false,
				Error:   "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// 获取用户信息
		meta := tokenBuilder.GetMeta()
		userInfo := UserInfo{
			UserID:   getString(meta, "user_id"),
			Username: getString(meta, "username"),
			Role:     getString(meta, "role"),
		}

		// 添加用户信息到日志上下文
		ctx = logger_tools.WithFields(ctx, map[string]any{
			"user_id":  userInfo.UserID,
			"username": userInfo.Username,
			"role":     userInfo.Role,
		})

		// 将上下文和用户信息存储到gin context
		c.Set("logger_ctx", ctx)
		c.Set("user_info", userInfo)
		c.Request = c.Request.WithContext(ctx)

		logger_tools.Info(ctx, "Request authenticated successfully")

		c.Next()
	}
}

// LoggerMiddleware 日志记录中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 创建日志上下文
		ctx := logger_tools.NewContext(c.Request.Context())
		ctx = logger_tools.WithFields(ctx, map[string]any{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
			"ip":     c.ClientIP(),
		})

		// 将上下文存储到gin context
		c.Set("logger_ctx", ctx)
		c.Request = c.Request.WithContext(ctx)

		logger_tools.Info(ctx, "Request started")

		c.Next()

		// 记录请求完成日志
		duration := time.Since(start)
		status := c.Writer.Status()

		ctx = logger_tools.WithFields(ctx, map[string]any{
			"status":   status,
			"duration": duration.String(),
		})

		if status >= 400 {
			logger_tools.Error(ctx, "Request completed with error")
		} else {
			logger_tools.Info(ctx, "Request completed successfully")
		}
	}
}

// GenerateToken 生成JWT token
func GenerateToken(userInfo UserInfo) (*TokenResponse, error) {
	config := jwt_tools.JwtConfig{
		SecretKey:     JWTSecretKey,
		SigningMethod: jwt.SigningMethodHS256,
		ExpireTime:    TokenExpire,
	}

	tokenBuilder := jwt_tools.NewTokenBuilder(config)

	meta := map[string]any{
		"user_id":  userInfo.UserID,
		"username": userInfo.Username,
		"role":     userInfo.Role,
	}

	tokenBuilder.SetMeta(meta)

	token, err := tokenBuilder.GenerateToken()
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		Token:     token,
		ExpiresAt: time.Now().Add(TokenExpire),
		UserInfo:  userInfo,
	}, nil
}

// ValidateUser 验证用户凭据（简单实现，实际应用中应连接数据库）
func ValidateUser(username, password string) (*UserInfo, error) {
	// 这里是简单的硬编码验证，实际应用中应该连接数据库验证
	if username == "admin" && password == "admin123" {
		return &UserInfo{
			UserID:   "1",
			Username: "admin",
			Role:     "admin",
		}, nil
	}

	// 可以添加更多用户
	if username == "user" && password == "user123" {
		return &UserInfo{
			UserID:   "2",
			Username: "user",
			Role:     "user",
		}, nil
	}

	return nil, ErrInvalidCredentials
}

// getString 安全地从map中获取字符串值
func getString(meta map[string]any, key string) string {
	if val, ok := meta[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/otkinlife/crud-generator/database"
	"github.com/otkinlife/crud-generator/middleware"
	"github.com/otkinlife/crud-generator/models"
	"github.com/otkinlife/crud-generator/services"
	"github.com/otkinlife/crud-generator/types"
	"github.com/otkinlife/go_tools/jwt_tools"
	"github.com/otkinlife/go_tools/logger_tools"
)

type APIServer struct {
	router        *gin.Engine
	configService *services.ConfigService
	crudService   *services.CRUDService
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

func NewAPIServer() *APIServer {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	server := &APIServer{
		router:        router,
		configService: services.NewConfigService(),
		crudService:   services.NewCRUDService(),
	}

	server.setupRoutes()
	return server
}

func (s *APIServer) setupRoutes() {
	// 添加日志中间件到所有路由
	s.router.Use(middleware.LoggerMiddleware())

	// 启用CORS
	s.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// 认证相关的公开路由（不需要JWT认证）
	auth := s.router.Group("/auth")
	{
		auth.POST("/login", s.login)
		auth.POST("/refresh", s.refreshToken)
	}

	// API路由组，需要JWT认证
	api := s.router.Group("/api")
	api.Use(middleware.JWTAuth()) // 应用JWT认证中间件
	{
		// 数据库连接信息（只读）
		api.GET("/connections", s.listConnections)
		api.POST("/connections/:id/test", s.testConnection)
		api.POST("/connections/:id/test-table", s.testConnectionWithTable)

		// 表配置管理 - 必须在CRUD路由之前，因为更具体
		configs := api.Group("/configs")
		{
			configs.GET("", s.listConfigs)
			configs.POST("", s.createConfig)
			configs.GET("/by-name/:name", s.getConfigByName)
			configs.GET("/:id", s.getConfig)
			configs.PUT("/:id", s.updateConfig)
			configs.DELETE("/:id", s.deleteConfig)
			configs.POST("/:id/test", s.testConfigConnection)
		}

		// CRUD API路由 - /api/{config_name}/{action} - 这些要放在configs后面
		api.GET("/:config_name/list", s.crudList)
		api.POST("/:config_name/create", s.crudCreate)
		api.PUT("/:config_name/update/:id", s.crudUpdate)
		api.DELETE("/:config_name/delete/:id", s.crudDelete)
		api.GET("/:config_name/dict/:field", s.crudDict)
	}

	// 静态文件和主页（不需要认证）
	s.router.Static("/webui", "./webui")
	s.router.GET("/", s.serveIndex)
	s.router.GET("/crud/:config_name", s.serveCRUDPage)
	s.router.NoRoute(s.serveIndex)
}

// 认证处理器
func (s *APIServer) login(c *gin.Context) {
	ctx, _ := c.Get("logger_ctx")
	logCtx := ctx.(context.Context)

	var req middleware.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger_tools.Warn(logCtx, "Invalid login request:", err.Error())
		c.JSON(http.StatusBadRequest, middleware.AuthResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	logger_tools.Info(logCtx, "Login attempt for user:", req.Username)

	// 验证用户凭据
	userInfo, err := middleware.ValidateUser(req.Username, req.Password)
	if err != nil {
		logger_tools.Warn(logCtx, "Login failed for user:", req.Username, "error:", err.Error())
		c.JSON(http.StatusUnauthorized, middleware.AuthResponse{
			Success: false,
			Error:   "Invalid username or password",
		})
		return
	}

	// 生成token
	tokenResponse, err := middleware.GenerateToken(*userInfo)
	if err != nil {
		logger_tools.Error(logCtx, "Failed to generate token:", err.Error())
		c.JSON(http.StatusInternalServerError, middleware.AuthResponse{
			Success: false,
			Error:   "Failed to generate token",
		})
		return
	}

	logger_tools.Info(logCtx, "Login successful for user:", req.Username)

	c.JSON(http.StatusOK, middleware.AuthResponse{
		Success: true,
		Data:    tokenResponse,
		Message: "Login successful",
	})
}

func (s *APIServer) refreshToken(c *gin.Context) {
	ctx, _ := c.Get("logger_ctx")
	logCtx := ctx.(context.Context)

	// 从header获取当前token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		logger_tools.Warn(logCtx, "Missing Authorization header for token refresh")
		c.JSON(http.StatusUnauthorized, middleware.AuthResponse{
			Success: false,
			Error:   "Missing authorization header",
		})
		return
	}

	tokenParts := strings.SplitN(authHeader, " ", 2)
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		logger_tools.Warn(logCtx, "Invalid authorization format for token refresh")
		c.JSON(http.StatusUnauthorized, middleware.AuthResponse{
			Success: false,
			Error:   "Invalid authorization format",
		})
		return
	}

	// 验证旧token并获取用户信息
	config := jwt_tools.JwtConfig{
		SecretKey:     middleware.JWTSecretKey,
		SigningMethod: jwt.SigningMethodHS256,
		ExpireTime:    middleware.TokenExpire,
	}

	tokenBuilder := jwt_tools.NewTokenBuilder(config)
	tokenBuilder.SetToken(tokenParts[1])

	// 即使token过期也要获取用户信息（用于刷新）
	if err := tokenBuilder.VerifyToken(); err != nil {
		logger_tools.Warn(logCtx, "Token verification failed during refresh:", err.Error())
		c.JSON(http.StatusUnauthorized, middleware.AuthResponse{
			Success: false,
			Error:   "Invalid token",
		})
		return
	}

	// 获取用户信息
	meta := tokenBuilder.GetMeta()
	userInfo := middleware.UserInfo{
		UserID:   getString(meta, "user_id"),
		Username: getString(meta, "username"),
		Role:     getString(meta, "role"),
	}

	logger_tools.Info(logCtx, "Token refresh for user:", userInfo.Username)

	// 生成新token
	tokenResponse, err := middleware.GenerateToken(userInfo)
	if err != nil {
		logger_tools.Error(logCtx, "Failed to generate new token:", err.Error())
		c.JSON(http.StatusInternalServerError, middleware.AuthResponse{
			Success: false,
			Error:   "Failed to generate new token",
		})
		return
	}

	logger_tools.Info(logCtx, "Token refresh successful for user:", userInfo.Username)

	c.JSON(http.StatusOK, middleware.AuthResponse{
		Success: true,
		Data:    tokenResponse,
		Message: "Token refreshed successfully",
	})
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

// 数据库连接信息处理器（只读）
func (s *APIServer) listConnections(c *gin.Context) {
	connections, err := s.configService.GetAvailableConnections()
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    connections,
	})
}

func (s *APIServer) testConnection(c *gin.Context) {
	connectionID := c.Param("id")

	if err := s.configService.TestConnection(connectionID); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Connection test successful",
	})
}

func (s *APIServer) testConnectionWithTable(c *gin.Context) {
	connectionID := c.Param("id")

	// 从请求体获取表名
	var request struct {
		TableName string `json:"table_name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "table_name is required",
		})
		return
	}

	if err := s.configService.TestConnectionWithTable(connectionID, request.TableName); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Table '%s' access test successful", request.TableName),
	})
}

func (s *APIServer) testConfigConnection(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid configuration ID",
		})
		return
	}

	if err := s.configService.TestConfigConnection(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Configuration connection and table test successful",
	})
}

// 表配置管理处理器
func (s *APIServer) listConfigs(c *gin.Context) {
	connectionID := c.Query("connection_id")

	configs, err := s.configService.GetConfigs(connectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    configs,
	})
}

func (s *APIServer) createConfig(c *gin.Context) {
	var config models.TableConfiguration
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := s.configService.CreateConfig(&config); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    config,
		Message: "Configuration created successfully",
	})
}

func (s *APIServer) getConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid ID",
		})
		return
	}

	config, err := s.configService.GetConfigByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    config,
	})
}

func (s *APIServer) getConfigByName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Configuration name is required",
		})
		return
	}

	config, err := s.configService.GetConfigByName(name)
	if err != nil {
		c.JSON(http.StatusNotFound, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    config,
	})
}

func (s *APIServer) updateConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid ID",
		})
		return
	}

	var config models.TableConfiguration
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := s.configService.UpdateConfig(uint(id), &config); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Configuration updated successfully",
	})
}

func (s *APIServer) deleteConfig(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid ID",
		})
		return
	}

	if err := s.configService.DeleteConfig(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: "Configuration deleted successfully",
	})
}

func (s *APIServer) serveIndex(c *gin.Context) {
	c.File("./webui/index.html")
}

func (s *APIServer) serveCRUDPage(c *gin.Context) {
	c.File("./webui/crud.html")
}

// CRUD API处理器
func (s *APIServer) crudList(c *gin.Context) {
	configName := c.Param("config_name")

	// 解析查询参数
	params := &types.QueryParams{}

	// 分页参数
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			params.Page = page
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil {
			params.PageSize = pageSize
		}
	}

	// 设置默认值
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	// 搜索参数 - 从查询参数中解析
	searchParams := make(map[string]interface{})
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 && !contains([]string{"page", "page_size", "sort", "order"}, key) {
			searchParams[key] = values[0]
		}
	}
	if len(searchParams) > 0 {
		params.Search = searchParams
	}

	// 排序参数
	if sortField := c.Query("sort"); sortField != "" {
		sortOrder := types.SortOrderASC
		if order := c.Query("order"); order != "" && order == "desc" {
			sortOrder = types.SortOrderDESC
		}
		params.Sort = []types.SortField{
			{
				Field: sortField,
				Order: sortOrder,
			},
		}
	}

	result, err := s.crudService.List(configName, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    result,
	})
}

func (s *APIServer) crudCreate(c *gin.Context) {
	configName := c.Param("config_name")

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON data: " + err.Error(),
		})
		return
	}

	result, err := s.crudService.Create(configName, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if !result.Success {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Data:    result,
		})
		return
	}

	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    result,
	})
}

func (s *APIServer) crudUpdate(c *gin.Context) {
	configName := c.Param("config_name")
	idStr := c.Param("id")

	// 尝试转换ID为整数，如果失败就作为字符串处理
	var id interface{}
	if idInt, err := strconv.Atoi(idStr); err == nil {
		id = idInt
	} else {
		id = idStr
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid JSON data: " + err.Error(),
		})
		return
	}

	result, err := s.crudService.Update(configName, id, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if !result.Success {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Data:    result,
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    result,
	})
}

func (s *APIServer) crudDelete(c *gin.Context) {
	configName := c.Param("config_name")
	idStr := c.Param("id")

	// 尝试转换ID为整数，如果失败就作为字符串处理
	var id interface{}
	if idInt, err := strconv.Atoi(idStr); err == nil {
		id = idInt
	} else {
		id = idStr
	}

	result, err := s.crudService.Delete(configName, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    result,
	})
}

func (s *APIServer) crudDict(c *gin.Context) {
	configName := c.Param("config_name")
	field := c.Param("field")

	result, err := s.crudService.GetDict(configName, field)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    result,
	})
}

// 辅助函数
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (s *APIServer) Start(port int) error {
	return s.router.Run(":" + strconv.Itoa(port))
}

func main() {
	// 初始化数据库管理器
	dbManager := database.GetDatabaseManager()
	if err := dbManager.InitMainDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	server := NewAPIServer()

	log.Println("Starting CRUD Generator Web UI on :8080")
	if err := server.Start(8080); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

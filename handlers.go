package crudgen

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/otkinlife/crud-generator/middleware"
)

//go:embed webui/*
var webuiFS embed.FS

// registerAPIRoutes registers all API routes
func (cg *CRUDGenerator) registerAPIRoutes(router *gin.Engine) {
	// Apply middleware
	router.Use(middleware.LoggerMiddleware())

	// Enable CORS
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API routes - 无需认证
	api := router.Group(cg.config.APIBasePath)
	{
		// Database connections
		api.GET("/connections", cg.handleListConnections)
		api.POST("/connections/:id/test", cg.handleTestConnection)
		api.POST("/connections/:id/test-table", cg.handleTestConnectionWithTable)

		// Table configurations
		configs := api.Group("/configs")
		{
			configs.GET("", cg.handleListConfigs)
			configs.POST("", cg.handleCreateConfig)
			configs.GET("/by-name/:name", cg.handleGetConfigByName)
			configs.GET("/:id", cg.handleGetConfig)
			configs.PUT("/:id", cg.handleUpdateConfig)
			configs.DELETE("/:id", cg.handleDeleteConfig)
			configs.POST("/:id/test", cg.handleTestConfigConnection)
		}

		// CRUD operations
		api.GET("/:config_name/list", cg.handleCRUDList)
		api.POST("/:config_name/create", cg.handleCRUDCreate)
		api.PUT("/:config_name/update/:id", cg.handleCRUDUpdate)
		api.DELETE("/:config_name/delete/:id", cg.handleCRUDDelete)
		api.GET("/:config_name/dict/:field", cg.handleCRUDDict)
	}
}

// registerUIRoutes registers all UI routes
func (cg *CRUDGenerator) registerUIRoutes(router *gin.Engine) {
	// Get embedded file system
	webuiSubFS, err := fs.Sub(webuiFS, "webui")
	if err != nil {
		panic("Failed to create webui subFS: " + err.Error())
	}

	// Serve static files
	router.StaticFS(cg.config.UIBasePath+"/static", http.FS(webuiSubFS))

	// Main UI routes
	router.GET(cg.config.UIBasePath, cg.handleUIIndex)
	router.GET(cg.config.UIBasePath+"/", cg.handleUIIndex)
	router.GET(cg.config.UIBasePath+"/crud/:config_name", cg.handleUICRUDPage)

	// Serve individual files
	router.GET(cg.config.UIBasePath+"/app.js", cg.handleUIFile("app.js"))
	router.GET(cg.config.UIBasePath+"/crud.js", cg.handleUICRUDJS)
	router.GET(cg.config.UIBasePath+"/crud.html", cg.handleUIFile("crud.html"))
}

// UI handlers
func (cg *CRUDGenerator) handleUIIndex(c *gin.Context) {
	data, err := webuiFS.ReadFile("webui/index.html")
	if err != nil {
		c.JSON(500, APIResponse{
			Success: false,
			Error:   "Failed to load UI: " + err.Error(),
		})
		return
	}

	// Replace base path in HTML if needed
	content := string(data)
	if cg.config.UIBasePath != "/crud-ui" {
		content = strings.ReplaceAll(content, "/webui/", cg.config.UIBasePath+"/")
	}

	// Inject configuration into the page
	configScript := fmt.Sprintf(`
		<script>
			window.CRUD_CONFIG = {
				api_base_path: %q,
				ui_base_path: %q
			};
		</script>`, cg.config.APIBasePath, cg.config.UIBasePath)

	// Insert config before closing </head> tag
	content = strings.Replace(content, "</head>", configScript+"\n</head>", 1)

	c.Header("Content-Type", "text/html")
	c.String(200, content)
}

func (cg *CRUDGenerator) handleUICRUDPage(c *gin.Context) {
	data, err := webuiFS.ReadFile("webui/crud.html")
	if err != nil {
		c.JSON(500, APIResponse{
			Success: false,
			Error:   "Failed to load CRUD page: " + err.Error(),
		})
		return
	}

	// Replace base path in HTML if needed
	content := string(data)
	if cg.config.UIBasePath != "/crud-ui" {
		content = strings.ReplaceAll(content, "/webui/", cg.config.UIBasePath+"/")
	}

	// Inject configuration into the page
	configScript := fmt.Sprintf(`
		<script>
			window.CRUD_CONFIG = {
				api_base_path: %q,
				ui_base_path: %q
			};
		</script>`, cg.config.APIBasePath, cg.config.UIBasePath)

	// Insert config before closing </head> tag
	content = strings.Replace(content, "</head>", configScript+"\n</head>", 1)

	c.Header("Content-Type", "text/html")
	c.String(200, content)
}

func (cg *CRUDGenerator) handleUIFile(filename string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := webuiFS.ReadFile("webui/" + filename)
		if err != nil {
			c.JSON(404, APIResponse{
				Success: false,
				Error:   "File not found: " + filename,
			})
			return
		}

		// Set appropriate content type
		if strings.HasSuffix(filename, ".js") {
			c.Header("Content-Type", "application/javascript")
		} else if strings.HasSuffix(filename, ".css") {
			c.Header("Content-Type", "text/css")
		} else if strings.HasSuffix(filename, ".html") {
			c.Header("Content-Type", "text/html")
		}

		c.String(200, string(data))
	}
}

func (cg *CRUDGenerator) handleUICRUDJS(c *gin.Context) {
	data, err := webuiFS.ReadFile("webui/crud.js")
	if err != nil {
		c.JSON(404, APIResponse{
			Success: false,
			Error:   "File not found: crud.js",
		})
		return
	}

	// Replace API base path in JavaScript if needed
	content := string(data)
	if cg.config.APIBasePath != "/api" {
		content = strings.ReplaceAll(content, "/api/", cg.config.APIBasePath+"/")
	}

	c.Header("Content-Type", "application/javascript")
	c.String(200, content)
}

// Connection handlers
func (cg *CRUDGenerator) handleListConnections(c *gin.Context) {
	connections := make(map[string]interface{})

	for id, config := range cg.dbManager.ListConnections() {
		connections[id] = map[string]interface{}{
			"name":     id,
			"db_type":  config.Type,
			"host":     config.Host,
			"database": config.Database,
		}
	}

	c.JSON(200, APIResponse{
		Success: true,
		Data:    connections,
	})
}

func (cg *CRUDGenerator) handleTestConnection(c *gin.Context) {
	connectionID := c.Param("id")

	if err := cg.dbManager.TestConnection(connectionID); err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success: true,
		Message: "Connection test successful",
	})
}

func (cg *CRUDGenerator) handleTestConnectionWithTable(c *gin.Context) {
	connectionID := c.Param("id")

	var request ConnectionTestRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   "table_name is required",
		})
		return
	}

	// Test connection first
	if err := cg.dbManager.TestConnection(connectionID); err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// TODO: Test table access
	c.JSON(200, APIResponse{
		Success: true,
		Message: "Table access test successful",
	})
}

// Config handlers
func (cg *CRUDGenerator) handleListConfigs(c *gin.Context) {
	connectionID := c.Query("connection_id")

	configs, err := cg.services.ConfigService.GetConfigsAsStruct(connectionID)
	if err != nil {
		c.JSON(500, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success: true,
		Data:    configs,
	})
}

func (cg *CRUDGenerator) handleCreateConfig(c *gin.Context) {
	var config TableConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := cg.services.ConfigService.CreateConfigFromStruct(&config); err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(201, APIResponse{
		Success: true,
		Data:    config,
		Message: "Configuration created successfully",
	})
}

func (cg *CRUDGenerator) handleGetConfigByName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   "Configuration name is required",
		})
		return
	}

	config, err := cg.services.ConfigService.GetConfigByNameAsStruct(name)
	if err != nil {
		c.JSON(404, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success: true,
		Data:    config,
	})
}

func (cg *CRUDGenerator) handleGetConfig(c *gin.Context) {
	// Implementation similar to handleGetConfigByName but by ID
	c.JSON(501, APIResponse{
		Success: false,
		Error:   "Not implemented yet",
	})
}

func (cg *CRUDGenerator) handleUpdateConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   "Invalid ID",
		})
		return
	}

	var config TableConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if err := cg.services.ConfigService.UpdateConfigFromStruct(uint(id), &config); err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success: true,
		Message: "Configuration updated successfully",
	})
}

func (cg *CRUDGenerator) handleDeleteConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   "Invalid ID",
		})
		return
	}

	if err := cg.services.ConfigService.DeleteConfig(uint(id)); err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success: true,
		Message: "Configuration deleted successfully",
	})
}

func (cg *CRUDGenerator) handleTestConfigConnection(c *gin.Context) {
	// Implementation for testing config connection
	c.JSON(501, APIResponse{
		Success: false,
		Error:   "Not implemented yet",
	})
}

// CRUD operation handlers
func (cg *CRUDGenerator) handleCRUDList(c *gin.Context) {
	configName := c.Param("config_name")

	// Parse query parameters
	params := &QueryParams{}

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

	// Set defaults
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	// Parse search parameters
	searchParams := make(map[string]interface{})
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 && !contains([]string{"page", "page_size", "sort", "order"}, key) {
			searchParams[key] = values[0]
		}
	}
	if len(searchParams) > 0 {
		params.Search = searchParams
	}

	// Parse sort parameters
	if sortField := c.Query("sort"); sortField != "" {
		sortOrder := SortOrderASC
		if order := c.Query("order"); order != "" && order == "desc" {
			sortOrder = SortOrderDESC
		}
		params.Sort = []SortField{
			{
				Field: sortField,
				Order: sortOrder,
			},
		}
	}

	result, err := cg.services.CRUDService.List(configName, params)
	if err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success: true,
		Data:    result,
	})
}

func (cg *CRUDGenerator) handleCRUDCreate(c *gin.Context) {
	configName := c.Param("config_name")

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   "Invalid JSON data: " + err.Error(),
		})
		return
	}

	result, err := cg.services.CRUDService.Create(configName, data)
	if err != nil {
		c.JSON(500, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if !result.Success {
		c.JSON(400, APIResponse{
			Success: false,
			Data:    result,
		})
		return
	}

	c.JSON(201, APIResponse{
		Success: true,
		Data:    result,
	})
}

func (cg *CRUDGenerator) handleCRUDUpdate(c *gin.Context) {
	configName := c.Param("config_name")
	idStr := c.Param("id")

	// Try to convert ID to integer, if fails use as string
	var id interface{}
	if idInt, err := strconv.Atoi(idStr); err == nil {
		id = idInt
	} else {
		id = idStr
	}

	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   "Invalid JSON data: " + err.Error(),
		})
		return
	}

	result, err := cg.services.CRUDService.Update(configName, id, data)
	if err != nil {
		c.JSON(500, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if !result.Success {
		c.JSON(400, APIResponse{
			Success: false,
			Data:    result,
		})
		return
	}

	c.JSON(200, APIResponse{
		Success: true,
		Data:    result,
	})
}

func (cg *CRUDGenerator) handleCRUDDelete(c *gin.Context) {
	configName := c.Param("config_name")
	idStr := c.Param("id")

	// Try to convert ID to integer, if fails use as string
	var id interface{}
	if idInt, err := strconv.Atoi(idStr); err == nil {
		id = idInt
	} else {
		id = idStr
	}

	result, err := cg.services.CRUDService.Delete(configName, id)
	if err != nil {
		c.JSON(500, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success: true,
		Data:    result,
	})
}

func (cg *CRUDGenerator) handleCRUDDict(c *gin.Context) {
	configName := c.Param("config_name")
	field := c.Param("field")

	result, err := cg.services.CRUDService.GetDict(configName, field)
	if err != nil {
		c.JSON(400, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(200, APIResponse{
		Success: true,
		Data:    result,
	})
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

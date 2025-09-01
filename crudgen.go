// Package crudgen provides a CRUD generator that can be embedded into Go applications
// It offers both programmatic API and embeddable web UI for database CRUD operations
package crudgen

import (
	"database/sql"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CRUDGenerator is the main interface for the CRUD generator
type CRUDGenerator struct {
	config    *Config
	dbManager *DatabaseManager
	services  *Services
}

// Config holds the configuration for the CRUD generator
type Config struct {
	// Database configuration
	DatabaseConfig map[string]DatabaseConnection `json:"databases"`

	// UI settings
	UIEnabled  bool   `json:"ui_enabled"`
	UIBasePath string `json:"ui_base_path"`

	// API settings
	APIBasePath string `json:"api_base_path"`
}

// DatabaseConnection represents a database connection configuration
type DatabaseConnection struct {
	Type         string `json:"type"` // postgresql, mysql
	Host         string `json:"host"`
	Port         int    `json:"port"`
	Database     string `json:"database"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	SSLMode      string `json:"ssl_mode"`
	MaxIdleConns int    `json:"max_idle_conns"`
	MaxOpenConns int    `json:"max_open_conns"`
}

// TableConfig represents configuration for a specific table
type TableConfig struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	TableName    string `json:"table_name"`
	ConnectionID string `json:"connection_id"`

	// SQL Schema
	CreateStatement string `json:"create_statement"`

	// Query configuration
	QueryPagination     bool   `json:"query_pagination"`
	QueryDisplayFields  string `json:"query_display_fields"`
	QuerySearchFields   string `json:"query_search_fields"`
	QuerySortableFields string `json:"query_sortable_fields"`

	// Create/Update configuration
	CreateCreatableFields string `json:"create_creatable_fields"`
	CreateValidationRules string `json:"create_validation_rules"`
	CreateDefaultValues   string `json:"create_default_values"`
	UpdateUpdatableFields string `json:"update_updatable_fields"`
	UpdateValidationRules string `json:"update_validation_rules"`

	// Other settings
	Description string `json:"description"`
	Tags        string `json:"tags"`
	IsActive    bool   `json:"is_active"`
	Version     int    `json:"version"`
}

// Services holds all the service instances
type Services struct {
	ConfigService *ConfigService
	CRUDService   *CRUDService
}

// DatabaseManager manages database connections
type DatabaseManager struct {
	connections map[string]*gorm.DB
	config      map[string]DatabaseConnection
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		UIEnabled:      true,
		UIBasePath:     "/crud-ui",
		APIBasePath:    "/api",
		DatabaseConfig: make(map[string]DatabaseConnection),
	}
}

// New creates a new CRUD generator instance
func New(config *Config) (*CRUDGenerator, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Initialize database manager
	dbManager, err := NewDatabaseManager(config.DatabaseConfig)
	if err != nil {
		return nil, err
	}

	// Initialize services
	services := &Services{
		ConfigService: NewConfigService(dbManager),
		CRUDService:   NewCRUDService(dbManager),
	}

	return &CRUDGenerator{
		config:    config,
		dbManager: dbManager,
		services:  services,
	}, nil
}

// NewWithGormDB creates a new CRUD generator instance using an existing GORM database connection
func NewWithGormDB(db *gorm.DB, connectionName string, config *Config) (*CRUDGenerator, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// If DatabaseConfig is empty, initialize it
	if config.DatabaseConfig == nil {
		config.DatabaseConfig = make(map[string]DatabaseConnection)
	}

	// Add the provided connection to config if not present
	if _, exists := config.DatabaseConfig[connectionName]; !exists {
		// Extract database info from the GORM DB connection
		dbConfig, err := extractDatabaseConfig(db)
		if err != nil {
			// If extraction fails, use a placeholder
			dbConfig = DatabaseConnection{
				Type: "unknown",
			}
		}
		config.DatabaseConfig[connectionName] = dbConfig
	}

	// Initialize database manager with existing connection
	dbManager := &DatabaseManager{
		connections: map[string]*gorm.DB{connectionName: db},
		config:      config.DatabaseConfig,
	}

	// Initialize services
	services := &Services{
		ConfigService: NewConfigService(dbManager),
		CRUDService:   NewCRUDService(dbManager),
	}

	return &CRUDGenerator{
		config:    config,
		dbManager: dbManager,
		services:  services,
	}, nil
}

// NewWithSQLDB creates a new CRUD generator instance using an existing sql.DB connection
func NewWithSQLDB(sqlDB *sql.DB, dbType, connectionName string, config *Config) (*CRUDGenerator, error) {
	// This would convert sql.DB to gorm.DB and then call NewWithGormDB
	// Implementation depends on the database type (postgres, mysql, etc.)
	panic("Not implemented yet - will be added in next iteration")
}

// RegisterRoutes registers all CRUD routes to the given gin router
func (cg *CRUDGenerator) RegisterRoutes(router *gin.Engine) {
	cg.registerAPIRoutes(router)
	if cg.config.UIEnabled {
		cg.registerUIRoutes(router)
	}
}

// RegisterAPIRoutes registers only the API routes (without UI)
func (cg *CRUDGenerator) RegisterAPIRoutes(router *gin.Engine) {
	cg.registerAPIRoutes(router)
}

// RegisterUIRoutes registers only the UI routes (without API)
func (cg *CRUDGenerator) RegisterUIRoutes(router *gin.Engine) {
	cg.registerUIRoutes(router)
}

// GetAPIHandler returns a http.Handler for the API routes only
func (cg *CRUDGenerator) GetAPIHandler() http.Handler {
	router := gin.New()
	cg.registerAPIRoutes(router)
	return router
}

// GetUIHandler returns a http.Handler for the UI routes only
func (cg *CRUDGenerator) GetUIHandler() http.Handler {
	router := gin.New()
	cg.registerUIRoutes(router)
	return router
}

// GetFullHandler returns a http.Handler with both API and UI routes
func (cg *CRUDGenerator) GetFullHandler() http.Handler {
	router := gin.New()
	cg.RegisterRoutes(router)
	return router
}

// AddTableConfig adds a new table configuration programmatically
func (cg *CRUDGenerator) AddTableConfig(config *TableConfig) error {
	return cg.services.ConfigService.CreateConfigFromStruct(config)
}

// GetTableConfig retrieves a table configuration by name
func (cg *CRUDGenerator) GetTableConfig(name string) (*TableConfig, error) {
	return cg.services.ConfigService.GetConfigByNameAsStruct(name)
}

// ListTableConfigs returns all table configurations
func (cg *CRUDGenerator) ListTableConfigs() ([]*TableConfig, error) {
	return cg.services.ConfigService.GetConfigsAsStruct("")
}

// UpdateTableConfig updates an existing table configuration
func (cg *CRUDGenerator) UpdateTableConfig(id uint, config *TableConfig) error {
	return cg.services.ConfigService.UpdateConfigFromStruct(id, config)
}

// DeleteTableConfig deletes a table configuration
func (cg *CRUDGenerator) DeleteTableConfig(id uint) error {
	return cg.services.ConfigService.DeleteConfig(id)
}

// CRUD operations - these provide direct programmatic access to CRUD operations

// List performs a list operation on the specified table
func (cg *CRUDGenerator) List(configName string, params *QueryParams) (*QueryResult, error) {
	return cg.services.CRUDService.List(configName, params)
}

// Create creates a new record in the specified table
func (cg *CRUDGenerator) Create(configName string, data map[string]interface{}) (*CRUDResult, error) {
	return cg.services.CRUDService.Create(configName, data)
}

// Update updates a record in the specified table
func (cg *CRUDGenerator) Update(configName string, id interface{}, data map[string]interface{}) (*CRUDResult, error) {
	return cg.services.CRUDService.Update(configName, id, data)
}

// Delete deletes a record from the specified table
func (cg *CRUDGenerator) Delete(configName string, id interface{}) (*CRUDResult, error) {
	return cg.services.CRUDService.Delete(configName, id)
}

// GetDict retrieves dictionary data for a field
func (cg *CRUDGenerator) GetDict(configName, field string) ([]DictItem, error) {
	return cg.services.CRUDService.GetDict(configName, field)
}

// Close closes all database connections
func (cg *CRUDGenerator) Close() error {
	return cg.dbManager.Close()
}

// extractDatabaseConfig extracts database configuration from a GORM DB connection
func extractDatabaseConfig(db *gorm.DB) (DatabaseConnection, error) {
	config := DatabaseConnection{}

	// Get the underlying sql.DB
	sqlDB, err := db.DB()
	if err != nil {
		return config, err
	}

	// Get the driver name to determine database type
	dialector := db.Dialector
	driverName := dialector.Name()

	switch driverName {
	case "postgres":
		config.Type = "postgresql"
	case "mysql":
		config.Type = "mysql"
	case "sqlite":
		config.Type = "sqlite"
	default:
		config.Type = driverName
	}

	// Try to extract connection details from the data source name if possible
	// This is database-specific and may not always work
	if dsn := getDSNFromDialector(dialector); dsn != "" {
		extractedConfig := parseDSN(dsn, config.Type)
		if extractedConfig.Host != "" {
			config.Host = extractedConfig.Host
		}
		if extractedConfig.Port != 0 {
			config.Port = extractedConfig.Port
		}
		if extractedConfig.Database != "" {
			config.Database = extractedConfig.Database
		}
		if extractedConfig.Username != "" {
			config.Username = extractedConfig.Username
		}
		if extractedConfig.SSLMode != "" {
			config.SSLMode = extractedConfig.SSLMode
		}
	}

	// Get connection pool settings
	stats := sqlDB.Stats()
	config.MaxOpenConns = stats.MaxOpenConnections
	config.MaxIdleConns = stats.Idle

	return config, nil
}

// getDSNFromDialector attempts to get the DSN from various GORM dialectors
func getDSNFromDialector(dialector gorm.Dialector) string {
	// This uses type assertion to get DSN from different dialectors
	// Each dialector type stores DSN differently
	switch dialector.Name() {
	case "postgres":
		// For postgres dialector, try to access DSN field via interface
		return ""
	case "mysql":
		// For mysql dialector, try to access DSN field via interface
		return ""
	case "sqlite":
		// For sqlite dialector, try to access DSN field via interface
		return ""
	default:
		// Generic fallback - may not work for all dialectors
		return ""
	}
}

// parseDSN parses a database connection string based on the database type
func parseDSN(dsn, dbType string) DatabaseConnection {
	config := DatabaseConnection{}

	switch dbType {
	case "postgresql":
		config = parsePostgresDSN(dsn)
	case "mysql":
		config = parseMySQLDSN(dsn)
	case "sqlite":
		config = parseSQLiteDSN(dsn)
	}

	return config
}

// parsePostgresDSN parses PostgreSQL connection string
func parsePostgresDSN(dsn string) DatabaseConnection {
	config := DatabaseConnection{}

	// PostgreSQL DSN can be in various formats:
	// "postgres://user:password@host:port/dbname?sslmode=disable"
	// "host=localhost port=5432 user=user password=password dbname=db sslmode=disable"

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		// URL format
		if u, err := url.Parse(dsn); err == nil {
			config.Host = u.Hostname()
			if port := u.Port(); port != "" {
				if p, err := strconv.Atoi(port); err == nil {
					config.Port = p
				}
			}
			config.Database = strings.TrimPrefix(u.Path, "/")
			if u.User != nil {
				config.Username = u.User.Username()
				config.Password, _ = u.User.Password()
			}
			if sslmode := u.Query().Get("sslmode"); sslmode != "" {
				config.SSLMode = sslmode
			}
		}
	} else {
		// Key-value format
		pairs := strings.Fields(dsn)
		for _, pair := range pairs {
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				key, value := kv[0], kv[1]
				switch key {
				case "host":
					config.Host = value
				case "port":
					if p, err := strconv.Atoi(value); err == nil {
						config.Port = p
					}
				case "user":
					config.Username = value
				case "password":
					config.Password = value
				case "dbname":
					config.Database = value
				case "sslmode":
					config.SSLMode = value
				}
			}
		}
	}

	return config
}

// parseMySQLDSN parses MySQL connection string
func parseMySQLDSN(dsn string) DatabaseConnection {
	config := DatabaseConnection{}

	// MySQL DSN format: [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
	// Example: "user:password@tcp(localhost:3306)/dbname"

	parts := strings.SplitN(dsn, "/", 2)
	if len(parts) == 2 {
		config.Database = strings.SplitN(parts[1], "?", 2)[0]

		// Parse the part before the database name
		beforeDB := parts[0]

		// Extract address part
		if strings.Contains(beforeDB, "(") && strings.Contains(beforeDB, ")") {
			start := strings.Index(beforeDB, "(")
			end := strings.Index(beforeDB, ")")
			address := beforeDB[start+1 : end]

			if strings.Contains(address, ":") {
				hostPort := strings.SplitN(address, ":", 2)
				config.Host = hostPort[0]
				if p, err := strconv.Atoi(hostPort[1]); err == nil {
					config.Port = p
				}
			} else {
				config.Host = address
			}

			beforeDB = beforeDB[:start]
		}

		// Extract username and password
		if strings.Contains(beforeDB, "@") {
			userPart := strings.TrimSuffix(beforeDB, "@")
			if strings.Contains(userPart, ":") {
				userPass := strings.SplitN(userPart, ":", 2)
				config.Username = userPass[0]
				config.Password = userPass[1]
			} else {
				config.Username = userPart
			}
		}
	}

	return config
}

// parseSQLiteDSN parses SQLite connection string
func parseSQLiteDSN(dsn string) DatabaseConnection {
	config := DatabaseConnection{}
	config.Database = dsn // For SQLite, the DSN is typically just the file path
	return config
}

// Internal methods are implemented in handlers.go

package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/otkinlife/crud-generator/models"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ConnectionPool struct {
	connections map[string]*gorm.DB
	mu          sync.RWMutex
}

type DatabaseManager struct {
	mainDB         *gorm.DB
	pool           *ConnectionPool
	dbConfigs      models.DatabaseConfigs
	configFilePath string
	memoryConfigs  models.DatabaseConfigs // 内存中的配置（如从GORM DB解析的）
}

var (
	instance *DatabaseManager
	once     sync.Once
)

func GetDatabaseManager() *DatabaseManager {
	once.Do(func() {
		// 从环境变量获取configs目录路径，默认为"./configs"
		configDir := os.Getenv("CONFIG_PATH")
		if configDir == "" {
			configDir = "./configs"
		}

		instance = &DatabaseManager{
			pool: &ConnectionPool{
				connections: make(map[string]*gorm.DB),
			},
			configFilePath: filepath.Join(configDir, "db.json"),
			memoryConfigs:  make(models.DatabaseConfigs),
		}
	})
	return instance
}

func (dm *DatabaseManager) LoadConfigs() error {
	// 初始化配置为空map
	dm.dbConfigs = make(models.DatabaseConfigs)

	// 首先尝试从文件加载配置
	data, err := os.ReadFile(dm.configFilePath)
	if err == nil {
		// 文件存在，解析JSON配置
		var fileConfigs models.DatabaseConfigs
		if err := json.Unmarshal(data, &fileConfigs); err != nil {
			return fmt.Errorf("failed to parse database config: %w", err)
		}
		// 合并文件配置
		for key, config := range fileConfigs {
			dm.dbConfigs[key] = config
		}
	}
	// 文件不存在时不报错，继续处理内存配置

	// 合并内存中的配置（来自NewWithGormDB等方法）
	if dm.memoryConfigs != nil {
		for key, config := range dm.memoryConfigs {
			// 内存配置优先级较高，会覆盖同名的文件配置
			dm.dbConfigs[key] = config
		}
	}

	// 如果既没有文件配置也没有内存配置，返回错误
	if len(dm.dbConfigs) == 0 {
		return fmt.Errorf("no database configurations found (neither file nor memory)")
	}

	return nil
}

// SetMemoryConfig 设置内存中的数据库配置
func (dm *DatabaseManager) SetMemoryConfig(connectionID string, config *models.DatabaseConfig) {
	if dm.memoryConfigs == nil {
		dm.memoryConfigs = make(models.DatabaseConfigs)
	}
	dm.memoryConfigs[connectionID] = config
}

// SetExistingConnection 设置一个已存在的GORM连接和其配置
func (dm *DatabaseManager) SetExistingConnection(connectionID string, db *gorm.DB, config *models.DatabaseConfig) {
	dm.pool.mu.Lock()
	defer dm.pool.mu.Unlock()

	// 设置连接
	dm.pool.connections[connectionID] = db

	// 设置内存配置
	if dm.memoryConfigs == nil {
		dm.memoryConfigs = make(models.DatabaseConfigs)
	}
	dm.memoryConfigs[connectionID] = config
}

func (dm *DatabaseManager) InitMainDB() error {
	if err := dm.LoadConfigs(); err != nil {
		return fmt.Errorf("failed to load configs: %w", err)
	}

	// 使用"default"连接作为主数据库
	defaultConfig, exists := dm.dbConfigs["default"]
	if !exists {
		return fmt.Errorf("default database connection not found in config")
	}

	dsn, err := dm.buildDSN(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to build default DSN: %w", err)
	}

	var dialector gorm.Dialector
	switch defaultConfig.DbType {
	case "postgresql":
		dialector = postgres.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	default:
		return fmt.Errorf("unsupported database type: %s", defaultConfig.DbType)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to main database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(defaultConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(defaultConfig.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(defaultConfig.ConnMaxLifetime) * time.Second)

	dm.mainDB = db

	// 自动迁移
	if err := dm.AutoMigrate(); err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}

	return nil
}

func (dm *DatabaseManager) AutoMigrate() error {
	return dm.mainDB.AutoMigrate(
		&models.TableConfiguration{},
	)
}

func (dm *DatabaseManager) GetMainDB() *gorm.DB {
	return dm.mainDB
}

func (dm *DatabaseManager) GetConnection(connectionID string) (*gorm.DB, error) {
	dm.pool.mu.RLock()
	if db, exists := dm.pool.connections[connectionID]; exists {
		dm.pool.mu.RUnlock()
		return db, nil
	}
	dm.pool.mu.RUnlock()

	// 连接不存在，创建新连接
	return dm.createConnection(connectionID)
}

func (dm *DatabaseManager) createConnection(connectionID string) (*gorm.DB, error) {
	dm.pool.mu.Lock()
	defer dm.pool.mu.Unlock()

	// 双重检查
	if db, exists := dm.pool.connections[connectionID]; exists {
		return db, nil
	}

	// 重新加载配置（支持热更新）
	if err := dm.LoadConfigs(); err != nil {
		return nil, fmt.Errorf("failed to reload configs: %w", err)
	}

	// 从JSON配置获取连接配置
	dbConfig, exists := dm.dbConfigs[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection '%s' not found in configuration", connectionID)
	}

	// 构建DSN
	dsn, err := dm.buildDSN(dbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	// 创建数据库连接
	var dialector gorm.Dialector
	switch dbConfig.DbType {
	case "postgresql":
		dialector = postgres.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbConfig.DbType)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(dbConfig.ConnMaxLifetime) * time.Second)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 缓存连接
	dm.pool.connections[connectionID] = db

	return db, nil
}

func (dm *DatabaseManager) buildDSN(dbConfig *models.DatabaseConfig) (string, error) {
	switch dbConfig.DbType {
	case "postgresql":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			dbConfig.Host, dbConfig.Port, dbConfig.Username, dbConfig.Password, dbConfig.DatabaseName, dbConfig.SSLMode)

		// 添加额外的连接参数
		for key, value := range dbConfig.ConnectionParams {
			dsn += fmt.Sprintf(" %s=%v", key, value)
		}

		return dsn, nil

	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			dbConfig.Username, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.DatabaseName)

		// 添加默认参数
		params := map[string]interface{}{
			"charset":   "utf8mb4",
			"parseTime": "True",
			"loc":       "Local",
		}

		// 合并额外参数
		for key, value := range dbConfig.ConnectionParams {
			params[key] = value
		}

		// 构建参数字符串
		var paramPairs []string
		for key, value := range params {
			paramPairs = append(paramPairs, fmt.Sprintf("%s=%v", key, value))
		}

		if len(paramPairs) > 0 {
			dsn += "?" + strings.Join(paramPairs, "&")
		}

		return dsn, nil

	default:
		return "", fmt.Errorf("unsupported database type: %s", dbConfig.DbType)
	}
}

func (dm *DatabaseManager) TestConnection(connectionID string) error {
	return dm.TestConnectionWithTable(connectionID, "")
}

func (dm *DatabaseManager) TestConnectionWithTable(connectionID string, tableName string) error {
	// 重新加载配置
	if err := dm.LoadConfigs(); err != nil {
		return fmt.Errorf("failed to reload configs: %w", err)
	}

	dbConfig, exists := dm.dbConfigs[connectionID]
	if !exists {
		return fmt.Errorf("connection '%s' not found in configuration", connectionID)
	}

	dsn, err := dm.buildDSN(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to build DSN: %w", err)
	}

	var dialector gorm.Dialector
	switch dbConfig.DbType {
	case "postgresql":
		dialector = postgres.Open(dsn)
	case "mysql":
		dialector = mysql.Open(dsn)
	default:
		return fmt.Errorf("unsupported database type: %s", dbConfig.DbType)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	defer sqlDB.Close()

	// 如果提供了表名，测试表查询；否则只测试连接
	if tableName != "" {
		// 执行 SELECT * FROM table_name LIMIT 1 来测试表访问
		query := fmt.Sprintf("SELECT * FROM %s LIMIT 1", tableName)
		if err := db.Raw(query).Error; err != nil {
			return fmt.Errorf("failed to query table '%s': %w", tableName, err)
		}
	} else {
		// 原有的ping测试
		if err := sqlDB.Ping(); err != nil {
			return fmt.Errorf("failed to ping: %w", err)
		}
	}

	return nil
}

func (dm *DatabaseManager) GetAllConnectionIDs() []string {
	if err := dm.LoadConfigs(); err != nil {
		return nil
	}

	var ids []string
	for id := range dm.dbConfigs {
		ids = append(ids, id)
	}
	return ids
}

func (dm *DatabaseManager) GetConnectionConfig(connectionID string) (*models.DatabaseConfig, error) {
	if err := dm.LoadConfigs(); err != nil {
		return nil, err
	}

	config, exists := dm.dbConfigs[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection '%s' not found", connectionID)
	}

	return config, nil
}

func (dm *DatabaseManager) RemoveConnection(connectionID string) {
	dm.pool.mu.Lock()
	defer dm.pool.mu.Unlock()

	if db, exists := dm.pool.connections[connectionID]; exists {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
		delete(dm.pool.connections, connectionID)
	}
}

func (dm *DatabaseManager) RefreshConnection(connectionID string) error {
	dm.RemoveConnection(connectionID)
	_, err := dm.GetConnection(connectionID)
	return err
}

func (dm *DatabaseManager) RefreshAllConnections() error {
	dm.pool.mu.Lock()
	defer dm.pool.mu.Unlock()

	// 关闭所有现有连接
	for id, db := range dm.pool.connections {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
		delete(dm.pool.connections, id)
	}

	// 重新加载配置
	return dm.LoadConfigs()
}

func (dm *DatabaseManager) Close() error {
	dm.pool.mu.Lock()
	defer dm.pool.mu.Unlock()

	// 关闭所有连接池中的连接
	for _, db := range dm.pool.connections {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}

	// 关闭主数据库连接
	if dm.mainDB != nil {
		if sqlDB, err := dm.mainDB.DB(); err == nil {
			return sqlDB.Close()
		}
	}

	return nil
}

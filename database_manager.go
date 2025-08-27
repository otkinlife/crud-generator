package crudgen

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDatabaseManager creates a new database manager with the given configuration
func NewDatabaseManager(config map[string]DatabaseConnection) (*DatabaseManager, error) {
	dm := &DatabaseManager{
		connections: make(map[string]*gorm.DB),
		config:      config,
	}

	// Initialize all configured database connections
	for id, dbConfig := range config {
		db, err := dm.createConnection(dbConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database %s: %w", id, err)
		}
		dm.connections[id] = db
	}

	return dm, nil
}

// createConnection creates a GORM database connection from configuration
func (dm *DatabaseManager) createConnection(config DatabaseConnection) (*gorm.DB, error) {
	var dsn string
	var dialector gorm.Dialector

	switch config.Type {
	case "postgresql", "postgres":
		dsn = fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.Username, config.Password,
			config.Database, config.SSLMode,
		)
		dialector = postgres.Open(dsn)

	case "mysql":
		dsn = fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			config.Username, config.Password, config.Host, config.Port, config.Database,
		)
		dialector = mysql.Open(dsn)

	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}

	// Configure GORM logger
	gormConfig := &gorm.Config{
		Logger: logger.New(
			log.New(log.Writer(), "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold: 200 * time.Millisecond,
				LogLevel:      logger.Silent, // Change to logger.Info for debug
				Colorful:      true,
			},
		),
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if config.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	} else {
		sqlDB.SetMaxIdleConns(10)
	}

	if config.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		sqlDB.SetMaxOpenConns(100)
	}

	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// GetConnection returns a database connection by ID
func (dm *DatabaseManager) GetConnection(connectionID string) (*gorm.DB, error) {
	db, exists := dm.connections[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection %s not found", connectionID)
	}
	return db, nil
}

// AddConnection adds a new database connection at runtime
func (dm *DatabaseManager) AddConnection(id string, config DatabaseConnection) error {
	db, err := dm.createConnection(config)
	if err != nil {
		return err
	}
	
	dm.connections[id] = db
	dm.config[id] = config
	return nil
}

// TestConnection tests if a connection is working
func (dm *DatabaseManager) TestConnection(connectionID string) error {
	db, err := dm.GetConnection(connectionID)
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}

// GetConnectionInfo returns information about a connection
func (dm *DatabaseManager) GetConnectionInfo(connectionID string) (*DatabaseInfo, error) {
	db, err := dm.GetConnection(connectionID)
	if err != nil {
		return nil, err
	}

	config, exists := dm.config[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection config %s not found", connectionID)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Test connection
	pingErr := sqlDB.Ping()
	connected := pingErr == nil

	info := &DatabaseInfo{
		ID:        connectionID,
		Name:      connectionID, // Could be enhanced with a separate name field
		Type:      config.Type,
		Host:      config.Host,
		Database:  config.Database,
		Connected: connected,
		LastPing:  time.Now(),
	}

	if connected {
		// Get table count (this is database-specific and simplified)
		var count int64
		switch config.Type {
		case "postgresql", "postgres":
			db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&count)
		case "mysql":
			db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", config.Database).Scan(&count)
		}
		info.TableCount = int(count)
	}

	return info, nil
}

// ListConnections returns all configured connections
func (dm *DatabaseManager) ListConnections() map[string]DatabaseConnection {
	return dm.config
}

// Close closes all database connections
func (dm *DatabaseManager) Close() error {
	var lastErr error
	
	for id, db := range dm.connections {
		if sqlDB, err := db.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				lastErr = fmt.Errorf("failed to close connection %s: %w", id, err)
			}
		}
	}
	
	return lastErr
}

// GetMainDB returns the main database connection (first one configured)
func (dm *DatabaseManager) GetMainDB() (*gorm.DB, error) {
	if len(dm.connections) == 0 {
		return nil, fmt.Errorf("no database connections configured")
	}
	
	// Return the first configured connection
	for _, db := range dm.connections {
		return db, nil
	}
	
	return nil, fmt.Errorf("no database connections found")
}
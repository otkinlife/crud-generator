package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	crudgen "github.com/otkinlife/crud-generator"
)

func main() {
	// Create configuration
	config := &crudgen.Config{
		EnableAuth:    false, // Disable auth for testing
		UIEnabled:     true,
		UIBasePath:    "/admin",
		APIBasePath:   "/api/v1",
		DatabaseConfig: map[string]crudgen.DatabaseConnection{
			"main": {
				Type:         "postgresql",
				Host:         "localhost",
				Port:         5432,
				Database:     "test_crud_db",
				Username:     "postgres",
				Password:     "password",
				SSLMode:      "disable",
				MaxIdleConns: 10,
				MaxOpenConns: 100,
			},
		},
	}

	// Create CRUD generator
	generator, err := crudgen.New(config)
	if err != nil {
		log.Fatal("Failed to create CRUD generator:", err)
	}
	defer generator.Close()

	// Add a sample table configuration
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
				age INTEGER,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`,
		QueryPagination: true,
		Description:     "User management table",
		IsActive:        true,
		Version:         1,
	}

	if err := generator.AddTableConfig(userTableConfig); err != nil {
		log.Printf("Warning: Failed to add user table config: %v", err)
	}

	// Create router and register routes
	router := gin.Default()

	// Your existing routes
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Package Test Server",
			"admin_ui": config.UIBasePath,
			"api": config.APIBasePath,
		})
	})

	// Register CRUD routes
	generator.RegisterRoutes(router)

	// Start server
	log.Println("Package test server starting on :8081")
	log.Printf("Main page: http://localhost:8081/")
	log.Printf("CRUD Admin UI: http://localhost:8081%s", config.UIBasePath)
	log.Printf("API Endpoints: http://localhost:8081%s", config.APIBasePath)
	log.Fatal(http.ListenAndServe(":8081", router))
}
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
		UIEnabled:   true,
		UIBasePath:  "/admin/ui",
		APIBasePath: "/api/v1",
		DatabaseConfig: map[string]crudgen.DatabaseConnection{
			"main": {
				Type:         "postgresql",
				Host:         "nas.kcjia.cn",
				Port:         5432,
				Database:     "crud_generator",
				Username:     "kcjia",
				Password:     "kcjia321",
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

	// Create router and register routes
	router := gin.Default()

	// Your existing routes
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":  "Package Test Server",
			"admin_ui": config.UIBasePath,
			"api":      config.APIBasePath,
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

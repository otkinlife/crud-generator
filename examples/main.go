package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/otkinlife/crud-generator/builder"
	"github.com/otkinlife/crud-generator/database"
	"github.com/otkinlife/crud-generator/types"
)

func main() {
	// 初始化数据库管理器（基于JSON配置）
	// CONFIG_PATH环境变量设置configs目录路径，默认为"./configs"
	dbManager := database.GetDatabaseManager()

	if err := dbManager.InitMainDB(); err != nil {
		log.Fatal("Failed to initialize main database:", err)
	}

	// 使用配置ID创建CRUD构建器
	configID := uint(1) // 假设配置ID为1
	crudBuilder, err := builder.NewCRUDBuilderFromConfig(configID)
	if err != nil {
		log.Fatal("Failed to create CRUD builder:", err)
	}

	fmt.Println("=== CRUD Generator Example ===")
	fmt.Printf("Configuration ID: %d\n", crudBuilder.GetConfigID())
	fmt.Printf("Table: %s\n", crudBuilder.GetConfig().TableName)

	fmt.Println("\n1. Create a new record:")
	createData := map[string]interface{}{
		"name":   "John Doe",
		"email":  "john@example.com",
		"age":    30,
		"status": "active",
	}

	createResult, err := crudBuilder.Create(createData)
	if err != nil {
		fmt.Printf("Error creating record: %v\n", err)
	} else {
		resultJSON, _ := json.MarshalIndent(createResult, "", "  ")
		fmt.Printf("Create result: %s\n", resultJSON)
	}

	fmt.Println("\n2. Query records with search and pagination:")
	queryParams := types.QueryParams{
		Page:     1,
		PageSize: 10,
		Search: map[string]interface{}{
			"name":   "John",
			"status": "active",
		},
		Sort: []types.SortField{
			{Field: "created_at", Order: types.SortOrderDESC},
		},
	}

	queryResult, err := crudBuilder.Query(queryParams)
	if err != nil {
		fmt.Printf("Error querying records: %v\n", err)
	} else {
		resultJSON, _ := json.MarshalIndent(queryResult, "", "  ")
		fmt.Printf("Query result: %s\n", resultJSON)
	}

	fmt.Println("\n3. Update record:")
	updateData := map[string]interface{}{
		"name": "John Smith",
		"age":  31,
	}

	updateResult, err := crudBuilder.Update(1, updateData)
	if err != nil {
		fmt.Printf("Error updating record: %v\n", err)
	} else {
		resultJSON, _ := json.MarshalIndent(updateResult, "", "  ")
		fmt.Printf("Update result: %s\n", resultJSON)
	}

	fmt.Println("\n4. Table schema:")
	schema := crudBuilder.GetTableSchema()
	schemaJSON, _ := json.MarshalIndent(schema, "", "  ")
	fmt.Printf("Table schema: %s\n", schemaJSON)

	fmt.Println("\nExample completed!")

	// 演示连接信息
	fmt.Println("\n5. Available database connections:")
	connectionIDs := dbManager.GetAllConnectionIDs()
	for _, id := range connectionIDs {
		if config, err := dbManager.GetConnectionConfig(id); err == nil {
			fmt.Printf("- %s: %s (%s)\n", id, config.Name, config.DbType)
		}
	}
}

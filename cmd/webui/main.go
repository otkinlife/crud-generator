package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/otkinlife/crud-generator/database"
	"github.com/otkinlife/crud-generator/webui"
)

func main() {
	var (
		port = flag.Int("port", 8080, "Web UI server port")
	)
	flag.Parse()

	// 从CONFIG_PATH环境变量获取configs目录路径，默认为"./configs"
	configDir := os.Getenv("CONFIG_PATH")
	if configDir == "" {
		configDir = "./configs"
	}
	configPath := filepath.Join(configDir, "db.json")

	fmt.Printf("Starting CRUD Generator Configuration Manager...\n")
	fmt.Printf("Web UI available at: http://localhost:%d\n", *port)
	fmt.Printf("Config directory: %s\n", configDir)
	fmt.Printf("Database config file: %s\n", configPath)

	// 初始化数据库管理器
	dbManager := database.GetDatabaseManager()

	if err := dbManager.InitMainDB(); err != nil {
		log.Fatal("Failed to initialize main database:", err)
	}

	fmt.Printf("✅ Database connections loaded successfully\n")
	fmt.Printf("✅ Main database initialized\n")

	// 启动Web服务器
	server := webui.NewAPIServer()
	if err := server.Start(*port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

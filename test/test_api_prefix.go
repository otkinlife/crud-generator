package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	crudgen "github.com/otkinlife/crud-generator"
)

func main() {
	// 创建配置，测试 /base_table/ 前缀
	config := crudgen.DefaultConfig()
	config.UIBasePath = "/admin"
	config.APIBasePath = "/base_table" // 测试这个前缀

	// 创建一个测试路由器
	router := gin.Default()

	// 添加主页路由
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":    "Testing /base_table/ API prefix",
			"admin_ui":   config.UIBasePath,
			"api_prefix": config.APIBasePath,
		})
	})

	// 创建一个模拟的配置注入测试页面
	router.GET(config.UIBasePath, func(c *gin.Context) {
		// 模拟 handlers.go 中的配置注入逻辑
		html := `<!DOCTYPE html>
<html>
<head>
    <title>API Prefix Test - /base_table/</title>
</head>
<body>
    <h1>Testing API Prefix: /base_table/</h1>
    <div id="test-results"></div>
    
    <script>
        // 模拟配置注入
        window.CRUD_CONFIG = {
            api_base_path: "/base_table",
            ui_base_path: "/admin"
        };
        
        // 模拟前端的 ConfigManager
        const ConfigManager = {
            config: null,
            
            loadConfig() {
                if (window.CRUD_CONFIG) {
                    this.config = window.CRUD_CONFIG;
                    return Promise.resolve(this.config);
                }
                
                console.warn('No config injected, using defaults');
                this.config = {
                    api_base_path: '/api',
                    ui_base_path: '/crud-ui'
                };
                return Promise.resolve(this.config);
            },
            
            getApiUrl(path) {
                const basePath = this.config ? this.config.api_base_path : '/api';
                return basePath + (path.startsWith('/') ? path : '/' + path);
            }
        };
        
        // 测试API URL生成
        document.addEventListener('DOMContentLoaded', async function() {
            await ConfigManager.loadConfig();
            
            const testResults = document.getElementById('test-results');
            
            const testCases = [
                '/configs',
                '/connections', 
                '/test/list',
                'configs',
                'connections',
                'test/list'
            ];
            
            let html = '<h2>API URL Generation Test Results:</h2>';
            html += '<table border="1" style="border-collapse: collapse; width: 100%;">';
            html += '<tr><th>Input Path</th><th>Generated URL</th><th>Expected</th></tr>';
            
            testCases.forEach(path => {
                const result = ConfigManager.getApiUrl(path);
                const expected = '/base_table' + (path.startsWith('/') ? path : '/' + path);
                const isCorrect = result === expected ? '✅' : '❌';
                html += '<tr>';
                html += '<td>' + path + '</td>';
                html += '<td>' + result + '</td>';
                html += '<td>' + expected + ' ' + isCorrect + '</td>';
                html += '</tr>';
            });
            
            html += '</table>';
            
            html += '<h2>Injected Configuration:</h2>';
            html += '<pre>' + JSON.stringify(window.CRUD_CONFIG, null, 2) + '</pre>';
            
            testResults.innerHTML = html;
        });
    </script>
</body>
</html>`

		c.Header("Content-Type", "text/html")
		c.String(200, html)
	})

	// 模拟一些API端点来测试路由
	apiGroup := router.Group(config.APIBasePath)
	{
		apiGroup.GET("/configs", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "configs endpoint working with /base_table/ prefix",
				"path":    c.Request.URL.Path,
			})
		})

		apiGroup.GET("/connections", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "connections endpoint working with /base_table/ prefix",
				"path":    c.Request.URL.Path,
			})
		})

		apiGroup.GET("/test/list", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "test/list endpoint working with /base_table/ prefix",
				"path":    c.Request.URL.Path,
			})
		})
	}

	// 启动服务器
	log.Println("API Prefix Test Server starting on :8082")
	log.Printf("Main page: http://localhost:8082/")
	log.Printf("UI Test page: http://localhost:8082%s", config.UIBasePath)
	log.Printf("API endpoints under: %s", config.APIBasePath)
	log.Fatal(http.ListenAndServe(":8082", router))
}

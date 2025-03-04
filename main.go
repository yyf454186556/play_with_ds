package main

import (
	"log"
	"play_with_ds/db"
	"play_with_ds/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

var NormalMap = make(map[string][]*model.ChatCompletionMessage)

func setUp() {
	db.Setup()
}

func main() {
	setUp()
	ask20Service := service.NewAsk20Service()
	normalService := service.NewNormalService()
	dndService := service.NewDNDService()
	go ask20Service.Ask20()
	go dndService.DND()

	// 创建 Gin 实例
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:44445", "http://124.222.139.115:44445"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// WebSocket 路由
	r.GET("/ws/ask20", ask20Service.Ask20Socketfunc)
	r.GET("/ws/dnd", dndService.DNDSocketfunc)

	r.POST("/ask20", ask20Service.Ask20Handler)
	r.POST("/dnd", dndService.DNDHandler)
	r.POST("/dnd-add-role", dndService.DNDAddRole)

	// 1. 设置角色基础信息
	// 2. 根据ID查询历史对话信息
	// 3. 展示所有图片信息
	r.POST("/ds", normalService.NormalDSHandler)
	r.GET("/all-images", normalService.GetImageList)

	if err := r.Run(":44444"); err != nil {
		log.Fatal("服务器启动失败:", err)
	}
}

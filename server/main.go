package main

import (
	"log"

	"zhixuan/server/config"
	"zhixuan/server/database"
	"zhixuan/server/gateway"
	"zhixuan/server/kbindex"
	"zhixuan/server/memory"
	"zhixuan/server/router"
	"zhixuan/server/scheduler"
)

func main() {
	config.Load()

	if err := database.Init(); err != nil {
		log.Fatal("数据库初始化失败: ", err)
	}

	gateway.Init()
	gateway.Get().RestoreQQListeners()
	gateway.Get().RestoreWeChatListeners()

	kbindex.Init()
	memory.Init()
	scheduler.Init()

	// 注入回调：gateway 创建定时任务后通过 scheduler 注册（避免 gateway↔scheduler 循环依赖）
	gateway.RegisterScheduleFunc = scheduler.AddJob

	r := router.Setup()
	log.Println("服务启动在", config.ServerPort)
	if err := r.Run(config.ServerPort); err != nil {
		log.Fatal("服务启动失败: ", err)
	}
}

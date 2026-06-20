package main

import (
	"log"
	"qzone-history/internal/delivery/gui"
	"qzone-history/internal/infrastructure/config"
	"qzone-history/version"
)

func main() {
	log.Printf("qzone-history 当前版本号: %s", version.Version)

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	srv := gui.NewServer(cfg)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

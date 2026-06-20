package main

import (
	"context"
	"fmt"
	"os"
	"qzone-history/internal/infrastructure/config"
	"qzone-history/internal/infrastructure/persistence"
	"qzone-history/internal/infrastructure/qzone_api"
	"qzone-history/pkg/database"
	"qzone-history/pkg/database/sqlite"
)

func main() {
	cfg, _ := config.LoadConfig()
	db := sqlite.NewSQLiteDB()
	db.Connect(&database.Config{DBName: cfg.Database.DBName})
	defer db.Close()
	user, _ := persistence.NewUserRepository(db).GetLastLoginUser(context.Background())
	body, err := qzone_api.ProbeFeedOffset(user.Cookies, user.QQ, 0, 10)
	if err != nil {
		fmt.Println("ERR:", err)
		return
	}
	s := string(body)
	if len(s) > 500 {
		s = s[:500]
	}
	fmt.Println(s)
	os.WriteFile("debug_probe_now.txt", body, 0644)
}

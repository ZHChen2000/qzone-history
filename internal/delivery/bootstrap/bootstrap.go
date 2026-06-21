package bootstrap

import (
	"fmt"
	"qzone-history/internal/delivery/app"
	"qzone-history/internal/infrastructure/config"
	"qzone-history/internal/infrastructure/persistence"
	"qzone-history/internal/infrastructure/qzone_api"
	"qzone-history/internal/usecase"
	"qzone-history/pkg/database"
	"qzone-history/pkg/database/sqlite"
)

type Stack struct {
	App *app.App
	DB  database.Database
}

func Build(cfg *config.Config, dbPath string) (*Stack, error) {
	db := sqlite.NewSQLiteDB()
	if err := db.Connect(&database.Config{DBName: dbPath}); err != nil {
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	qzoneAPIClient := qzone_api.NewQzoneAPIClient(cfg)
	userRepo := persistence.NewUserRepository(db)
	momentRepo := persistence.NewMomentRepository(db)
	activityRepo := persistence.NewActivityRepository(db)
	boardMessageRepo := persistence.NewBoardMessageRepository(db)
	friendRepo := persistence.NewFriendRepository(db)

	authUseCase := usecase.NewAuthUseCase(qzoneAPIClient, userRepo)
	momentUseCase := usecase.NewMomentUseCase(momentRepo, qzoneAPIClient)
	boardMessageUseCase := usecase.NewBoardMessageUseCase(boardMessageRepo, qzoneAPIClient)
	friendUseCase := usecase.NewFriendUseCase(friendRepo)
	exportUseCase := usecase.NewExportUseCase(momentRepo, boardMessageRepo, friendRepo, activityRepo)
	activityUseCase := usecase.NewActivityUseCase(qzoneAPIClient, activityRepo)
	reconstructionUseCase := usecase.NewReconstructionUseCase(activityRepo, momentRepo, boardMessageRepo)

	application := app.NewApp(
		authUseCase, momentUseCase, boardMessageUseCase, friendUseCase,
		exportUseCase, activityUseCase, reconstructionUseCase,
	)

	return &Stack{App: application, DB: db}, nil
}

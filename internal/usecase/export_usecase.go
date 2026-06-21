// export_usecase.go

package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"qzone-history/internal/domain/entity"
	"qzone-history/internal/domain/repository"
	"qzone-history/internal/domain/usecase"
	"qzone-history/pkg/export"
	"qzone-history/pkg/paths"
	"time"
)

type exportUseCase struct {
	momentRepo       repository.MomentRepository
	boardMessageRepo repository.BoardMessageRepository
	friendRepo       repository.FriendRepository
	activityRepo     repository.ActivityRepository
}

func NewExportUseCase(
	momentRepo repository.MomentRepository,
	boardMessageRepo repository.BoardMessageRepository,
	friendRepo repository.FriendRepository,
	activityRepo repository.ActivityRepository,
) usecase.ExportUseCase {
	return &exportUseCase{
		momentRepo:       momentRepo,
		boardMessageRepo: boardMessageRepo,
		friendRepo:       friendRepo,
		activityRepo:     activityRepo,
	}
}

func (u *exportUseCase) ExportUserDataToJSON(ctx context.Context, userQQ string) error {
	moments, _ := u.momentRepo.FindByUserQQ(ctx, userQQ, -1, 0)
	boardMessages, _ := u.boardMessageRepo.FindByUserQQ(ctx, userQQ, -1, 0)
	friends, _ := u.friendRepo.FindFriendsByUserQQ(ctx, userQQ)
	export.SortMomentsDesc(moments)
	export.SortBoardDesc(boardMessages)

	exportData := struct {
		UserQQ        string                `json:"userQQ"`
		Moments       []entity.Moment       `json:"moments"`
		BoardMessages []entity.BoardMessage `json:"boardMessages"`
		Friends       []entity.Friend       `json:"friends"`
	}{
		UserQQ:        userQQ,
		Moments:       moments,
		BoardMessages: boardMessages,
		Friends:       friends,
	}

	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data to JSON: %w", err)
	}

	filename := paths.ExportJSONPath(userQQ)
	err = os.WriteFile(filename, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}

func (u *exportUseCase) ExportUserDataToExcel(ctx context.Context, userQQ string) error {
	return fmt.Errorf("ExportUserDataToExcel not implemented")
}

func (u *exportUseCase) ExportUserDataToHTML(ctx context.Context, userQQ string) error {
	moments, err := u.momentRepo.FindByUserQQ(ctx, userQQ, -1, 0)
	if err != nil {
		return fmt.Errorf("读取说说失败: %w", err)
	}
	boardMessages, err := u.boardMessageRepo.FindByUserQQ(ctx, userQQ, -1, 0)
	if err != nil {
		return fmt.Errorf("读取留言板失败: %w", err)
	}
	activities, err := u.activityRepo.FindByUserQQ(ctx, userQQ, -1, 0)
	if err != nil {
		return fmt.Errorf("读取活动记录失败: %w", err)
	}
	export.SortMomentsDesc(moments)
	export.SortBoardDesc(boardMessages)
	viewerActivities := export.FilterActivitiesForViewer(activities)
	export.SortActivitiesDesc(viewerActivities)

	filename := paths.ViewerHTMLPath(userQQ)
	return export.WriteViewerHTML(filename, export.ViewerPayload{
		UserQQ:        userQQ,
		GeneratedAt:   time.Now().Format("2006-01-02 15:04:05"),
		Moments:       moments,
		BoardMessages: boardMessages,
		Activities:    viewerActivities,
	})
}

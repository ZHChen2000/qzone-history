package usecase

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"qzone-history/internal/domain/entity"
	"qzone-history/internal/domain/repository"
	"qzone-history/internal/domain/usecase"
	"qzone-history/internal/infrastructure/qzone_api"
)

type activityUseCase struct {
	qzoneAPI     qzone_api.QzoneAPIClient
	activityRepo repository.ActivityRepository
}

func NewActivityUseCase(qzoneAPI qzone_api.QzoneAPIClient, activityRepo repository.ActivityRepository) usecase.ActivityUseCase {
	return &activityUseCase{
		qzoneAPI:     qzoneAPI,
		activityRepo: activityRepo,
	}
}

func (a *activityUseCase) GetActivities(ctx context.Context, userQQ string, limit, offset int) ([]entity.Activity, error) {
	return a.activityRepo.FindByUserQQ(ctx, userQQ, limit, offset)
}

func (a *activityUseCase) GetAllActivities(ctx context.Context, userQQ string) ([]entity.Activity, error) {
	return a.activityRepo.FindByUserQQ(ctx, userQQ, -1, 0)
}

func (a *activityUseCase) SaveActivity(ctx context.Context, activity entity.Activity) error {
	return a.activityRepo.Insert(ctx, activity)
}

func (a *activityUseCase) GetActivityCount(ctx context.Context, userQQ string) (int, error) {
	activities, err := a.activityRepo.FindByUserQQ(ctx, userQQ, -1, 0)
	if err != nil {
		return 0, err
	}
	return len(activities), nil
}

func (a *activityUseCase) GetActivitiesByType(ctx context.Context, activityType entity.ActivityType, limit, offset int) ([]entity.Activity, error) {
	return a.activityRepo.FindByType(ctx, activityType, limit, offset)
}
func (a *activityUseCase) generateActivityID(message *entity.Activity) string {
	data := fmt.Sprintf("%s%s%s", message.Content, message.Timestamp.String(), message.SenderQQ)
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}
func (a *activityUseCase) FetchActivities(ctx context.Context, user entity.User, maxOffset, targetYear int) ([]entity.Activity, error) {
	activitiesPtr, err := a.qzoneAPI.GetAllActivities(user.Cookies, qzone_api.FetchOptions{
		MaxOffset:  maxOffset,
		TargetYear: targetYear,
		Ctx:        ctx,
	})

	activities := make([]entity.Activity, 0, len(activitiesPtr))
	for i, actPtr := range activitiesPtr {
		activities = append(activities, *actPtr)
		activities[i].ID = a.generateActivityID(actPtr)
	}

	if len(activities) == 0 {
		if err != nil && errors.Is(err, context.Canceled) {
			return nil, err
		}
		if err != nil {
			return nil, fmt.Errorf("获取所有活动失败: %w", err)
		}
		return activities, nil
	}

	batchSize := 100
	for i := 0; i < len(activities); i += batchSize {
		if err := ctx.Err(); err != nil {
			return activities[:i], err
		}
		end := i + batchSize
		if end > len(activities) {
			end = len(activities)
		}
		batch := activities[i:end]
		if saveErr := a.activityRepo.BatchImport(ctx, batch); saveErr != nil {
			return nil, fmt.Errorf("保存活动批次 %d-%d 失败: %w", i, end, saveErr)
		}
	}

	if err != nil {
		return activities, err
	}
	return activities, nil
}

func (a *activityUseCase) FetchActivity(ctx context.Context, user entity.User, offset int) (entity.Activity, error) {
	activitiesPtr, err := a.qzoneAPI.GetActivities(user.Cookies, offset, 1)
	if err != nil {
		return entity.Activity{}, fmt.Errorf("获取活动失败: %w", err)
	}
	activities := make([]entity.Activity, len(activitiesPtr))
	for i, actPtr := range activitiesPtr {
		activities[i] = *actPtr
	}
	if len(activities) == 0 {
		return entity.Activity{}, fmt.Errorf("未找到活动")
	}
	activity := activities[0]
	err = a.activityRepo.Insert(ctx, activity)
	if err != nil {
		return entity.Activity{}, fmt.Errorf("保存活动失败: %w", err)
	}

	return activity, nil
}

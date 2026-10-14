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
	if ctx == nil {
		ctx = context.Background()
	}
	saved := make(map[string]struct{})
	streamingAttempted := false
	convert := func(items []*entity.Activity) []entity.Activity {
		result := make([]entity.Activity, 0, len(items))
		for _, item := range items {
			if item == nil {
				continue
			}
			copy := *item
			copy.ID = a.generateActivityID(item)
			result = append(result, copy)
		}
		return result
	}
	persist := func(items []*entity.Activity) error {
		streamingAttempted = true
		batch := convert(items)
		if len(batch) == 0 {
			return nil
		}
		if err := a.activityRepo.BatchImport(ctx, batch); err != nil {
			return err
		}
		for _, activity := range batch {
			saved[activity.ID] = struct{}{}
		}
		return nil
	}
	activitiesPtr, fetchErr := a.qzoneAPI.GetAllActivities(user.Cookies, qzone_api.FetchOptions{
		MaxOffset: maxOffset, TargetYear: targetYear, Ctx: ctx, OnBatch: persist,
	})
	activities := convert(activitiesPtr)
	if len(activities) == 0 {
		if fetchErr != nil && errors.Is(fetchErr, context.Canceled) {
			return nil, fetchErr
		}
		if fetchErr != nil {
			return nil, fmt.Errorf("获取所有活动失败: %w", fetchErr)
		}
		return activities, nil
	}
	if fetchErr != nil && streamingAttempted {
		return activities, fetchErr
	}
	// Existing clients that do not stream batches are still supported. Already
	// checkpointed activities are not written again at the end of a scan.
	pending := make([]entity.Activity, 0, len(activities))
	for _, activity := range activities {
		if _, ok := saved[activity.ID]; !ok {
			pending = append(pending, activity)
		}
	}
	for start := 0; start < len(pending); start += 100 {
		if err := ctx.Err(); err != nil {
			return activities, err
		}
		end := start + 100
		if end > len(pending) {
			end = len(pending)
		}
		if err := a.activityRepo.BatchImport(ctx, pending[start:end]); err != nil {
			return activities, fmt.Errorf("保存活动批次 %d-%d 失败: %w", start, end, err)
		}
	}
	return activities, fetchErr
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

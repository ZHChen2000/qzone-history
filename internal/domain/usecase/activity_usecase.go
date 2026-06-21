package usecase

import (
	"context"
	"qzone-history/internal/domain/entity"
)

type ActivityUseCase interface {
	GetActivities(ctx context.Context, userQQ string, offset, count int) ([]entity.Activity, error)

	GetAllActivities(ctx context.Context, userQQ string) ([]entity.Activity, error)

	SaveActivity(ctx context.Context, activity entity.Activity) error

	GetActivityCount(ctx context.Context, userQQ string) (int, error)

	GetActivitiesByType(ctx context.Context, activityType entity.ActivityType, limit, offset int) ([]entity.Activity, error)

	FetchActivities(ctx context.Context, user entity.User, maxOffset, targetYear int) ([]entity.Activity, error)

	FetchActivity(ctx context.Context, user entity.User, offset int) (entity.Activity, error)
}

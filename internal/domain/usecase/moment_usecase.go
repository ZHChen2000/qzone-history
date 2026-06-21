package usecase

import (
	"context"
	"qzone-history/internal/domain/entity"
)

type MomentUseCase interface {
	CreateMoment(ctx context.Context, moment *entity.Moment) error

	GetMomentsByUserQQ(ctx context.Context, userQQ string, limit, offset int) ([]entity.Moment, error)

	AddLikeToMoment(ctx context.Context, momentID string) error

	AddCommentToMoment(ctx context.Context, comment *entity.Comment) error

	IncrementMomentViews(ctx context.Context, momentID string) error

	MarkMomentAsDeleted(ctx context.Context, momentID string) error

	MarkMomentAsReconstructed(ctx context.Context, momentID string) error

	GetMomentByID(ctx context.Context, momentID string) (*entity.Moment, error)

	ImportVisibleMoments(ctx context.Context, user entity.User) (int, error)
}

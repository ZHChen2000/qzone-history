package usecase

import (
	"context"
	"qzone-history/internal/domain/entity"
)

type UserUseCase interface {
	GetUserInfo(ctx context.Context, userQQ string) (*entity.User, error)

	UpdateUserInfo(ctx context.Context, user *entity.User) error

	SaveUser(ctx context.Context, user *entity.User) error
}

package usecase

import (
	"context"
	"qzone-history/internal/domain/entity"
)

type FriendUseCase interface {
	AddFriend(ctx context.Context, friend *entity.Friend) error

	GetFriendsByUserQQ(ctx context.Context, userQQ string) ([]entity.Friend, error)

	GetFriend(ctx context.Context, userQQ, friendQQ string) (*entity.Friend, error)

	CheckFriendship(ctx context.Context, userQQ, friendQQ string) (bool, error)
}

package repository

import (
	"context"
	"qzone-history/internal/domain/entity"
)

type FriendRepository interface {
	BatchImport(ctx context.Context, friends []entity.Friend) error
	Insert(ctx context.Context, friend entity.Friend) error
	FindFriendsByUserQQ(ctx context.Context, userQQ string) ([]entity.Friend, error)
	FindFriend(ctx context.Context, userQQ, friendQQ string) (*entity.Friend, error)
	IsFriend(ctx context.Context, userQQ, friendQQ string) (bool, error)
}

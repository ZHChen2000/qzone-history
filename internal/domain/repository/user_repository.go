package repository

import (
	"context"
	"qzone-history/internal/domain/entity"
)

// UserRepository defines the interface for user storage, supporting saving, getting by QQ, and retrieving all users.
type UserRepository interface {
	Save(ctx context.Context, user entity.User) error
	FindByQQ(ctx context.Context, qq string) (*entity.User, error)
	GetLastLoginUser(ctx context.Context) (*entity.User, error)
	Update(ctx context.Context, user entity.User) error
	UpdateLoginStatus(ctx context.Context, qq string, status entity.LoginStatus) error
}

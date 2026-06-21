package usecase

import (
	"context"
	"qzone-history/internal/domain/entity"
)

type AuthUseCase interface {
	CheckLocalLoginStatus(ctx context.Context) (*entity.User, bool, error)

	GetLoginQRCode(ctx context.Context) ([]byte, string, error)

	CheckQRCodeLoginStatus(ctx context.Context, qrsig string) (entity.LoginStatus, string, error)

	CompleteLogin(ctx context.Context, loginResponse string) (*entity.User, error)

	RefreshLogin(ctx context.Context, user *entity.User) (*entity.User, error)

	Logout(ctx context.Context) error
}

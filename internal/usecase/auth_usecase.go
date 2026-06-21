package usecase

import (
	"context"
	"errors"
	"fmt"
	"qzone-history/internal/domain/entity"
	"qzone-history/internal/domain/repository"
	"qzone-history/internal/domain/usecase"
	"qzone-history/internal/infrastructure/qzone_api"
	"time"
)

type authUseCase struct {
	qzoneAPI qzone_api.QzoneAPIClient
	userRepo repository.UserRepository
}

func NewAuthUseCase(qzoneAPI qzone_api.QzoneAPIClient, userRepo repository.UserRepository) usecase.AuthUseCase {
	return &authUseCase{
		qzoneAPI: qzoneAPI,
		userRepo: userRepo,
	}
}

func (a *authUseCase) CheckLocalLoginStatus(ctx context.Context) (*entity.User, bool, error) {
	lastLoginUser, err := a.userRepo.GetLastLoginUser(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("获取最后登录用户失败: %w", err)
	}

	if lastLoginUser == nil {
		return nil, false, nil
	}

	if time.Now().After(lastLoginUser.LoginExpireTime) {
		return nil, false, nil
	}

	userInfo, err := a.qzoneAPI.GetUserInfo(lastLoginUser.Cookies)
	if err != nil {
		return nil, false, nil
	}

	lastLoginUser.Nickname = userInfo.Nickname
	lastLoginUser.AvatarURL = userInfo.AvatarURL
	err = a.userRepo.Update(ctx, *lastLoginUser)
	if err != nil {
		return nil, false, fmt.Errorf("更新用户信息失败: %w", err)
	}

	return lastLoginUser, true, nil
}

func (a *authUseCase) GetLoginQRCode(ctx context.Context) ([]byte, string, error) {
	return a.qzoneAPI.GetLoginQRCode()
}

func (a *authUseCase) CheckQRCodeLoginStatus(ctx context.Context, qrsig string) (entity.LoginStatus, string, error) {
	status, responseText, err := a.qzoneAPI.CheckLoginStatus(qrsig)
	return status, responseText, err
}

func (a *authUseCase) CompleteLogin(ctx context.Context, loginResponse string) (*entity.User, error) {
	cookies, err := a.qzoneAPI.CompleteLogin(loginResponse)
	if err != nil {
		return nil, fmt.Errorf("完成登录失败: %w", err)
	}

	userInfo, err := a.qzoneAPI.GetUserInfo(cookies)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	user := &entity.User{
		QQ:              userInfo.QQ,
		Nickname:        userInfo.Nickname,
		AvatarURL:       userInfo.AvatarURL,
		Cookies:         cookies,
		LoginExpireTime: time.Now().Add(24 * time.Hour),
		LoginStatus:     entity.LoginStatusSuccess,
	}

	err = a.userRepo.Update(ctx, *user)
	if err != nil {
		return nil, fmt.Errorf("保存用户信息失败: %w", err)
	}

	return user, nil
}

func (a *authUseCase) RefreshLogin(ctx context.Context, user *entity.User) (*entity.User, error) {
	userInfo, err := a.qzoneAPI.GetUserInfo(user.Cookies)
	if err != nil {
		return nil, errors.New("登录已过期，需要重新登录")
	}

	user.Nickname = userInfo.Nickname
	user.AvatarURL = userInfo.AvatarURL
	user.LoginExpireTime = time.Now().Add(24 * time.Hour)

	err = a.userRepo.Update(ctx, *user)
	if err != nil {
		return nil, fmt.Errorf("更新用户信息失败: %w", err)
	}

	return user, nil
}

func (a *authUseCase) Logout(ctx context.Context) error {
	currentUser, err := a.userRepo.GetLastLoginUser(ctx)
	if err != nil {
		return fmt.Errorf("获取当前登录用户失败: %w", err)
	}

	if currentUser == nil {
		return nil
	}

	currentUser.Cookies = nil
	currentUser.LoginExpireTime = time.Time{}

	err = a.userRepo.Update(ctx, *currentUser)
	if err != nil {
		return fmt.Errorf("更新用户信息失败: %w", err)
	}

	return nil
}

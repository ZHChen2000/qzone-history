package usecase

import (
	"context"
	"fmt"
	"qzone-history/internal/domain/entity"
	"qzone-history/internal/domain/repository"
	"qzone-history/internal/domain/usecase"
	"qzone-history/internal/infrastructure/qzone_api"
)

type boardMessageUseCase struct {
	boardMessageRepo repository.BoardMessageRepository
	qzoneAPI         qzone_api.QzoneAPIClient
}

func NewBoardMessageUseCase(repo repository.BoardMessageRepository, qzoneAPI qzone_api.QzoneAPIClient) usecase.BoardMessageUseCase {
	return &boardMessageUseCase{
		boardMessageRepo: repo,
		qzoneAPI:         qzoneAPI,
	}
}

func (u *boardMessageUseCase) CreateBoardMessage(ctx context.Context, message *entity.BoardMessage) error {
	return u.boardMessageRepo.Insert(ctx, *message)
}

func (u *boardMessageUseCase) GetBoardMessagesByUserQQ(ctx context.Context, userQQ string, limit, offset int) ([]entity.BoardMessage, error) {
	return u.boardMessageRepo.FindByUserQQ(ctx, userQQ, limit, offset)
}

func (u *boardMessageUseCase) GetBoardMessageByID(ctx context.Context, messageID string) (*entity.BoardMessage, error) {
	return u.boardMessageRepo.FindByID(ctx, messageID)
}

func (u *boardMessageUseCase) ImportBoardMessages(ctx context.Context, user entity.User) (int, error) {
	messages, err := u.qzoneAPI.GetBoardMessages(user.Cookies)
	if err != nil {
		return 0, fmt.Errorf("获取留言板失败: %w", err)
	}
	if err := u.boardMessageRepo.DeleteByUserQQ(ctx, user.QQ); err != nil {
		return 0, fmt.Errorf("清理旧留言板数据失败: %w", err)
	}
	for _, message := range messages {
		if err := u.boardMessageRepo.Insert(ctx, message); err != nil {
			return len(messages), err
		}
	}
	return len(messages), nil
}

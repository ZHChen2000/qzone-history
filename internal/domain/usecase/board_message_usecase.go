package usecase

import (
	"context"
	"qzone-history/internal/domain/entity"
)

type BoardMessageUseCase interface {
	CreateBoardMessage(ctx context.Context, message *entity.BoardMessage) error

	GetBoardMessagesByUserQQ(ctx context.Context, userQQ string, limit, offset int) ([]entity.BoardMessage, error)

	GetBoardMessageByID(ctx context.Context, messageID string) (*entity.BoardMessage, error)

	ImportBoardMessages(ctx context.Context, user entity.User) (int, error)
}

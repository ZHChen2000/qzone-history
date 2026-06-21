package repository

import (
	"context"
	"qzone-history/internal/domain/entity"
)

type BoardMessageRepository interface {
	BatchImport(ctx context.Context, messages []entity.BoardMessage) error
	Insert(ctx context.Context, message entity.BoardMessage) error
	FindByUserQQ(ctx context.Context, userQQ string, limit, offset int) ([]entity.BoardMessage, error)
	DeleteByUserQQ(ctx context.Context, userQQ string) error
	FindByID(ctx context.Context, messageID string) (*entity.BoardMessage, error)
}

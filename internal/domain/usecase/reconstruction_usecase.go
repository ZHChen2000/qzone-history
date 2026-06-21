package usecase

import (
	"context"
)

type ReconstructionUseCase interface {
	ReconstructMomentsFromActivities(ctx context.Context, userQQ string) error

	ReconstructBoardMessagesFromActivities(ctx context.Context, userQQ string) error
}

package usecase

import (
	"context"
)

type ExportUseCase interface {
	ExportUserDataToJSON(ctx context.Context, userQQ string) error

	ExportUserDataToExcel(ctx context.Context, userQQ string) error

	ExportUserDataToHTML(ctx context.Context, userQQ string) error
}

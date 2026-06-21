package usecase

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"qzone-history/internal/domain/entity"
	"qzone-history/internal/domain/repository"
	"qzone-history/internal/domain/usecase"
	"strings"
)

type reconstructionUseCase struct {
	activityRepo     repository.ActivityRepository
	momentRepo       repository.MomentRepository
	boardMessageRepo repository.BoardMessageRepository
}

func NewReconstructionUseCase(
	activityRepo repository.ActivityRepository,
	momentRepo repository.MomentRepository,
	boardMessageRepo repository.BoardMessageRepository,
) usecase.ReconstructionUseCase {
	return &reconstructionUseCase{
		activityRepo:     activityRepo,
		momentRepo:       momentRepo,
		boardMessageRepo: boardMessageRepo,
	}
}

func (u *reconstructionUseCase) ReconstructMomentsFromActivities(ctx context.Context, userQQ string) error {
	activities, err := u.activityRepo.FindByUserQQ(ctx, userQQ, -1, 0)
	if err != nil {
		return err
	}

	momentMap := make(map[string]*entity.Moment)

	for _, activity := range activities {
		if activity.Type == entity.TypeBoardMessage || activity.Type == entity.TypeBoardReply {
			continue
		}
		switch activity.Type {
		case entity.TypeMoment, entity.TypeLike, entity.TypeView, entity.TypeComment, entity.TypeForward:
		default:
			continue
		}

		momentKey := generateMomentKey(activity)
		moment, ok := momentMap[momentKey]
		if !ok {
			moment = &entity.Moment{
				ID:              momentKey,
				SenderQQ:        activity.SenderQQ,
				UserQQ:          activity.ReceiverQQ,
				Content:         activity.Content,
				Timestamp:       activity.Timestamp,
				TimeText:        activity.TimeText,
				ImageURLs:       activity.ImageURLs,
				IsReconstructed: true,
			}
			momentMap[momentKey] = moment
		}

		switch activity.Type {
		case entity.TypeMoment:
			updateMomentFromActivity(moment, activity)
		case entity.TypeLike:
			moment.Likes++
		case entity.TypeView:
			moment.Views++
		case entity.TypeComment:
			comment := reconstructCommentFromActivity(activity)
			moment.Comments = append(moment.Comments, comment)
		}
	}

	for _, moment := range momentMap {
		if err := u.momentRepo.UpsertMoment(ctx, *moment); err != nil {
			return err
		}
	}

	return nil
}

func generateMomentKey(activity entity.Activity) string {
	key := activity.Content + activity.ReceiverQQ
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

func updateMomentFromActivity(moment *entity.Moment, activity entity.Activity) {
	if len(activity.Content) > len(moment.Content) {
		moment.Content = activity.Content
	}
	if activity.Timestamp.Before(moment.Timestamp) || moment.Timestamp.IsZero() {
		moment.Timestamp = activity.Timestamp
		moment.TimeText = activity.TimeText
	}
	if len(activity.ImageURLs) > len(moment.ImageURLs) {
		moment.ImageURLs = activity.ImageURLs
	}
}

func reconstructCommentFromActivity(activity entity.Activity) entity.Comment {
	return entity.Comment{
		UserQQ:    activity.SenderQQ,
		Content:   activity.Content,
		Timestamp: activity.Timestamp,
		TimeText:  activity.TimeText,
	}
}

func (u *reconstructionUseCase) ReconstructBoardMessagesFromActivities(ctx context.Context, userQQ string) error {
	activities, err := u.activityRepo.FindByUserQQ(ctx, userQQ, -1, 0)
	if err != nil {
		return err
	}

	existing, err := u.boardMessageRepo.FindByUserQQ(ctx, userQQ, -1, 0)
	if err != nil {
		return err
	}

	apiKeys := make(map[string]struct{}, len(existing))
	for _, msg := range existing {
		apiKeys[boardDedupKey(msg)] = struct{}{}
	}

	added := 0
	for _, activity := range activities {
		if !isBoardActivity(activity) {
			continue
		}
		candidate := reconstructBoardMessageFromActivity(activity)
		key := boardDedupKey(candidate)
		if _, ok := apiKeys[key]; ok {
			continue
		}
		_ = candidate.BeforeCreate(nil)
		if err := u.boardMessageRepo.Insert(ctx, candidate); err != nil {
			return err
		}
		apiKeys[key] = struct{}{}
		added++
	}

	_ = added
	return nil
}

func isBoardActivity(a entity.Activity) bool {
	return a.Type == entity.TypeBoardMessage || a.Type == entity.TypeBoardReply
}

func reconstructBoardMessageFromActivity(activity entity.Activity) entity.BoardMessage {
	content := strings.TrimSpace(activity.Content)
	if activity.Type == entity.TypeBoardReply && content != "" {
		content = "[回复] " + content
	}
	return entity.BoardMessage{
		UserQQ:     activity.ReceiverQQ,
		SenderQQ:   activity.SenderQQ,
		SenderName: activity.SenderName,
		Content:    content,
		Timestamp:  activity.Timestamp,
		TimeText:   activity.TimeText,
	}
}

func boardDedupKey(m entity.BoardMessage) string {
	day := m.TimeText
	if !m.Timestamp.IsZero() {
		day = m.Timestamp.Format("2006-01-02")
	}
	norm := strings.TrimSpace(m.Content)
	if len(norm) > 80 {
		norm = norm[:80]
	}
	return m.SenderQQ + "|" + day + "|" + norm
}

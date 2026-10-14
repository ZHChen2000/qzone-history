package export

import (
	"qzone-history/internal/domain/entity"
	"reflect"
	"testing"
	"time"
)

func TestExportSortUsesAbsoluteTextAndIsStable(t *testing.T) {
	newer := time.Date(2020, 9, 30, 23, 59, 19, 0, time.FixedZone("Asia/Shanghai", 8*3600))
	moments := []entity.Moment{
		{ID: "unknown-1", TimeText: "9月6日 02:59"},
		{ID: "older", TimeText: "\\t2018年6月23日 09:18\\t", Content: "keep text"},
		{ID: "newer", Timestamp: newer, TimeText: "2017年1月1日"},
		{ID: "tie", TimeText: "2020年9月30日 23:59:19"},
		{ID: "unknown-2"},
	}
	boards := make([]entity.BoardMessage, len(moments))
	activities := make([]entity.Activity, len(moments))
	for i, m := range moments {
		boards[i] = entity.BoardMessage{ID: m.ID, Timestamp: m.Timestamp, TimeText: m.TimeText}
		activities[i] = entity.Activity{ID: m.ID, Timestamp: m.Timestamp, TimeText: m.TimeText}
	}
	SortMomentsDesc(moments)
	SortBoardDesc(boards)
	SortActivitiesDesc(activities)
	want := []string{"newer", "tie", "older", "unknown-1", "unknown-2"}
	for name, ids := range map[string][]string{
		"moments":    {moments[0].ID, moments[1].ID, moments[2].ID, moments[3].ID, moments[4].ID},
		"boards":     {boards[0].ID, boards[1].ID, boards[2].ID, boards[3].ID, boards[4].ID},
		"activities": {activities[0].ID, activities[1].ID, activities[2].ID, activities[3].ID, activities[4].ID},
	} {
		if !reflect.DeepEqual(ids, want) {
			t.Errorf("%s: %v", name, ids)
		}
	}
	if !moments[2].Timestamp.IsZero() || moments[2].TimeText != "\\t2018年6月23日 09:18\\t" || moments[2].Content != "keep text" {
		t.Fatal("sorting modified source fields")
	}
}

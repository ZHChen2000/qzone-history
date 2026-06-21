package export

import (
	"qzone-history/internal/domain/entity"
	"sort"
	"time"
)

func SortMomentsDesc(items []entity.Moment) {
	sort.Slice(items, func(i, j int) bool {
		return momentTime(items[i]).After(momentTime(items[j]))
	})
}

func SortBoardDesc(items []entity.BoardMessage) {
	sort.Slice(items, func(i, j int) bool {
		ti, tj := boardTime(items[i]), boardTime(items[j])
		if ti.IsZero() && tj.IsZero() {
			return false
		}
		if ti.IsZero() {
			return false
		}
		if tj.IsZero() {
			return true
		}
		return ti.After(tj)
	})
}

func SortActivitiesDesc(items []entity.Activity) {
	sort.Slice(items, func(i, j int) bool {
		return activityTime(items[i]).After(activityTime(items[j]))
	})
}

func FilterActivitiesForViewer(items []entity.Activity) []entity.Activity {
	out := make([]entity.Activity, 0, len(items))
	for _, a := range items {
		if a.Type == entity.TypeBoardMessage || a.Type == entity.TypeBoardReply {
			continue
		}
		out = append(out, a)
	}
	return out
}

func momentTime(m entity.Moment) time.Time {
	if !m.Timestamp.IsZero() {
		return m.Timestamp
	}
	return time.Time{}
}

func boardTime(m entity.BoardMessage) time.Time {
	if !m.Timestamp.IsZero() {
		return m.Timestamp
	}
	return time.Time{}
}

func activityTime(a entity.Activity) time.Time {
	if !a.Timestamp.IsZero() {
		return a.Timestamp
	}
	return time.Time{}
}

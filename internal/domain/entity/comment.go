package entity

import "time"

type Comment struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	MomentID  string    `json:"momentID" gorm:"index"`
	UserQQ    string    `json:"userQQ" gorm:"index"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	TimeText  string    `json:"timeText"`
}

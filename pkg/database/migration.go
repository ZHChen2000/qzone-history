package database

import (
	"qzone-history/internal/domain/entity"
)

func AutoMigrate(db Database) error {
	return db.DB().AutoMigrate(
		&entity.User{},
		&entity.Moment{},
		&entity.BoardMessage{},
		&entity.Activity{},
		&entity.Comment{},
		&entity.Friend{},
	)
}

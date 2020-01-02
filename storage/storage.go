package storage

import "github.com/swishcloud/filesync-web/storage/models"

type Storage interface {
	Commit()
	Rollback()
	InsertFile(md5, name, userId string)
	DeleteFile(id string)
	GetFile(id string) models.File
	GetAllFiles() []models.File
}

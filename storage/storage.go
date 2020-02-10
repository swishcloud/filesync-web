package storage

import (
	"github.com/swishcloud/filesync-web/storage/models"
)

type Storage interface {
	Commit()
	Rollback()
	InsertFile(name, userId, file_info_id string)
	InsertFileInfo(md5, name, userId, size string)
	DeleteFile(id string)
	GetFile(id string) models.File
	GetAllFiles() []models.File
	GetFileBlocks(server_file_id string) []models.FileBlock
	GetServerFileByFileId(file_id string) *models.ServerFile
	GetServerFile(md5, name string) *models.ServerFile
	CompleteServerFile(server_file_id string)
	AddFileBlock(server_file_id, name string, start, end int64)
	GetUserByOpId(op_id string) *models.User
	AddOrUpdateUser(sub string, name string)
}

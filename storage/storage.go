package storage

import (
	"github.com/swishcloud/filesync-web/storage/models"
)

type Storage interface {
	Commit() error
	Rollback() error
	GetFileByPath(path string, user_id string) map[string]interface{}
	GetFileInfo(md5 string) map[string]interface{}
	InsertFileInfo(md5, userId string, size int64)
	GetFile(id string) models.File
	GetFiles(p_id, partition_id string, revision int64) []models.File
	GetFileBlocks(server_file_id string) []models.FileBlock
	GetServerFileByFileId(file_id string) *models.ServerFile
	CompleteServerFile(server_file_id string)
	AddFileBlock(server_file_id, name string, start, end int64)
	GetUserByOpId(op_id string) *models.User
	AddOrUpdateUser(sub string, name string)
	GetServers() []models.Server
	GetServer(server_id string) *models.Server
	GetDirectory(path string, partition_id string, revision int64) *models.Directory
	AddServer(name, ip, port string)
	UpdateServer(id, name, ip, port string)
	DeleteServer(id string)
	SetFileHidden(file_id string, is_hidden bool)
	SuperDoFileActions(actions []Action, user_id, partition_id string) (err error)
	GetHistoryRevisions(file_id, partition_id string) []map[string]interface{}
}

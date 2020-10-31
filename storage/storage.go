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
	GetFileById(id string) models.File
	GetFiles(path string, commit_id string, max_commit_id string, partition_id string) (files []map[string]interface{}, err error)
	GetFileBlocks(server_file_id string) []models.FileBlock
	GetServerFileByFileId(file_id string) *models.ServerFile
	CompleteServerFile(server_file_id string)
	AddFileBlock(server_file_id, name string, start, end int64)
	GetUserByOpId(op_id string) *models.User
	AddOrUpdateUser(sub string, name string) (user *models.User, err error)
	GetServers() []models.Server
	GetServer(server_id string) *models.Server
	GetFile(path string, partition_id string, commit_id string, file_type int) map[string]interface{}
	AddServer(name, ip, port string)
	UpdateServer(id, name, ip, port string)
	DeleteServer(id string)
	SetFileHidden(file_id string, is_hidden bool)
	SuperDoFileActions(actions []Action, user_id, partition_id string) (err error)
	GetHistoryRevisions(path, partition_id string, max_revision int64) []map[string]interface{}
	GetExactFileByPath(path string, partition_id string) map[string]interface{}
	AddShare(path string, partition_id string, commit_id string, max_commit_id string, user_id string) (token string)
	GetShareByToken(token string) map[string]interface{}
	GetPartitionLatestCommit(partition_id string) map[string]interface{}
	GetPartitionFirstCommit(partition_id string) map[string]interface{}
	GetCommits(partition_id string, from_commit string) []map[string]interface{}
	GetCommitById(commit_id string) map[string]interface{}
	GetRecentCommits(partition_id string) []map[string]interface{}
	GetCommitChanges(partition_id string, commit_id string) []map[string]interface{}
	GetFilePath(partition_id string, id string, revision int64) string
}

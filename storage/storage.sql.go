package storage

import (
	"database/sql"
	"time"

	"github.com/google/uuid"

	"github.com/swishcloud/filesync-web/storage/models"
	"github.com/swishcloud/gostudy/tx"
	"github.com/swishcloud/identity-provider/global"
)

type SQLManager struct {
	Tx *tx.Tx
}

var db *sql.DB

func NewSQLManager(db_conn_info string) *SQLManager {
	if db == nil {
		d, err := sql.Open("postgres", db_conn_info)
		global.Err(err)
		db = d
	}
	tx, err := tx.NewTx(db)
	if err != nil {
		panic(err)
	}
	return &SQLManager{Tx: tx}
}
func (m *SQLManager) Commit() {
	m.Tx.Commit()
}
func (m *SQLManager) Rollback() {
	m.Tx.Rollback()
}
func (m *SQLManager) InsertFile(md5, name, userId string) {
	m.Tx.MustExec("INSERT INTO public.file( 	id, insert_time, md5, name, user_id) 	VALUES ($1, $2, $3, $4, $5);", uuid.New(), time.Now().UTC(), md5, name, userId)
}
func (m *SQLManager) DeleteFile(id string) {
	m.Tx.MustExec("DELETE FROM public.file WHERE id=$1;", id)

}
func (m *SQLManager) GetFile(id string) models.File {
	var sql = "SELECT id, insert_time, md5, name, user_id 	FROM public.file;"
	rows := m.Tx.MustQuery(sql)
	files := getFiles(rows)
	if len(files) == 1 {
		return files[0]
	} else {
		panic("not found")
	}
}
func (m *SQLManager) GetAllFiles() []models.File {
	var sql = "SELECT id, insert_time, md5, name, user_id 	FROM public.file;"
	rows := m.Tx.MustQuery(sql)
	files := getFiles(rows)
	return files
}
func getFiles(rows *tx.Rows) []models.File {
	files := []models.File{}
	for rows.Next() {
		file := &models.File{}
		rows.MustScan(file.Id, file.InsertTime, file.Md5, file.Name, file.User_id)
	}
	return files
}

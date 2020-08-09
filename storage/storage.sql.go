package storage

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	_ "github.com/lib/pq"
	"github.com/swishcloud/filesync-web/storage/models"
	"github.com/swishcloud/gostudy/tx"
)

type SQLManager struct {
	Tx *tx.Tx
}

var db *sql.DB

func NewSQLManager(db_conn_info string) *SQLManager {
	if db == nil {
		d, err := sql.Open("postgres", db_conn_info)
		if err != nil {
			log.Fatal(err)
		}
		db = d
	}
	tx, err := tx.NewTx(db)
	if err != nil {
		panic(err)
	}
	return &SQLManager{Tx: tx}
}
func (m *SQLManager) Commit() error {
	return m.Tx.Commit()
}
func (m *SQLManager) Rollback() error {
	return m.Tx.Rollback()
}
func (m *SQLManager) GetFileByPath(path string, user_id string) map[string]interface{} {
	query := `
WITH RECURSIVE CTE AS (
    SELECT *,'' as path,0 as level from file where name='' and user_id=$1
  UNION ALL
    SELECT f.*,CTE.path || '/' || f.name,CTE.level+1 from file f
	inner join CTE on f.p_id=CTE.id where f."end" is null and $2 like CTE.path || '/' || f.name || '%'
  )
SELECT * from CTE order by level desc limit 1;`
	sr := ScanRows(m.Tx.MustQuery(query, user_id, path))
	fmt.Println(sr)
	if len(sr) > 1 {
		panic("should only return one row data.")
	} else {
		return sr[0]
	}
}
func (m *SQLManager) DoFileActions(actions []models.FileAction, user_id string) {
	defer func() {
		err := recover()
		if err != nil {
			m.Rollback()
			panic(err)
		}
	}()
	m.Tx.MustExec(`LOCK TABLE file IN SHARE ROW EXCLUSIVE MODE;`)
	row := m.Tx.MustQueryRow(`select max(a."max") from (select max(start) from file union select max("end") from file) a;`)
	max := int64(-1)
	row.MustScan(&max)

	d := NewFileManager(m, user_id, max+1)
	for _, action := range actions {
		longest_folder := ""
		if action.Md5 == "" {
			longest_folder = action.Path
		} else {
			regexp, err := regexp.Compile(".+/")
			if err != nil {
				panic(err)
			}
			longest_folder = strings.TrimRight(regexp.FindString(action.Path), "/")
		}
		dir := d.makeDirAll(longest_folder)
		if action.Md5 != "" {
			name := filepath.Base(action.Path)
			d.insertFile(name, dir["id"].(string), action.Md5, false, 1)
		}
	}
}

type fileManager struct {
	m        *SQLManager
	user_id  string
	revision int64
}

func NewFileManager(m *SQLManager, user_id string, revision int64) *fileManager {
	d := new(fileManager)
	d.m = m
	d.user_id = user_id
	d.revision = revision
	return d
}

//insert a file if t value is 1,if t value is 2 insert a directory.
func (d *fileManager) insertFile(name, p_id, md5 string, is_hidden bool, t int) {
	if t != 1 && t != 2 {
		panic("the file type can only be 1 or 2.")
	}
	file_info_id := new(string)
	if md5 != "" {
		file_info := d.m.GetFileInfo(md5)
		if file_info == nil {
			panic("the file does not exits for md5 " + md5)
		}
		*file_info_id = file_info["id"].(string)
	} else {
		file_info_id = nil
	}

	if t == 1 {
		query := `select * from file where p_id=$1 and name=$2 and "end" is null`
		file := d.m.Tx.ScanRow(query, p_id, name)
		if file != nil {
			if file["file_info_id"].(string) != *file_info_id || file["is_hidden"].(string) != strconv.FormatBool(is_hidden) {
				d.m.Tx.MustExec(`update file set "end"=$1 where id=$2`, d.revision, file["id"].(string))
			} else {
				//the file already exists,just return.
				return
			}
		}
	}
	insert_file := `INSERT INTO public.file(
		id, insert_time, name, description, user_id, file_info_id,is_deleted,p_id,is_hidden,type,start)
		VALUES ($1, $2, $3, $4, $5, $6,$7,$8,$9,$10,$11);`
	id := uuid.New().String()
	var stored_p_id *string
	if p_id == "" || name == "" {
		if p_id != "" || name != "" || t != 2 {
			panic("parameters error.if add a root folder for a new user,then both p_id and name must be empty value.")
		}
		stored_p_id = nil
	} else {
		stored_p_id = &p_id
	}
	d.m.Tx.MustExec(insert_file, id, time.Now().UTC(), name, "", d.user_id, file_info_id, false, stored_p_id, is_hidden, t, strconv.FormatInt(d.revision, 10))
}
func (d *fileManager) makeDirAll(path string) map[string]interface{} {
	for {
		file := d.m.GetFileByPath(path, d.user_id)
		p := file["path"].(string)
		id := file["id"].(string)
		if p == "" {
			p = "/"
		}
		if p != path {
			fmt.Println("creating directory...")
			name := path[len(p):]
			regexp, err := regexp.Compile("[^/]+")
			if err != nil {
				panic(err)
			}
			name = regexp.FindString(name)
			d.insertFile(name, id, "", false, 2)
		} else {
			return file
		}
	}
}

func (m *SQLManager) GetServers() []models.Server {
	return m.getServers("")
}
func (m *SQLManager) GetFileInfo(md5 string) map[string]interface{} {
	query := `select a.*,b.id as server_file_id,b.uploaded_size,b.is_completed,b.server_id,c.ip,c.port from file_info a
	inner join server_file b
	on a.id=b.file_info_id 
	inner join server c
	on b.server_id=c.id where a.md5=$1`
	return m.Tx.ScanRow(query, md5)
}
func ScanRows(rows *tx.Rows) []map[string]interface{} {
	result := []map[string]interface{}{}
	columns, err := rows.ColumnTypes()
	args := make([]interface{}, len(columns))
	for i, _ := range args {
		zero_str_p := new(string)
		args[i] = &zero_str_p
	}
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		rows.MustScan(args...)
		m := map[string]interface{}{}
		for i, _ := range args {
			val := *args[i].(**string)
			if val == nil {
				m[columns[i].Name()] = nil
			} else {
				m[columns[i].Name()] = *val
			}
		}
		result = append(result, m)
	}
	rows.Close()
	return result
}
func (m *SQLManager) getServers(where string, args ...interface{}) []models.Server {
	query := `SELECT id, name, ip, port
	FROM public.server ` + where
	rows := m.Tx.MustQuery(query, args...)
	result := []models.Server{}
	for rows.Next() {
		server := models.Server{}
		err := rows.Scan(&server.Id, &server.Name, &server.Ip, &server.Port)
		if err != nil {
			panic(err)
		}
		result = append(result, server)
	}
	return result
}
func (m *SQLManager) GetServer(server_id string) *models.Server {
	servers := m.getServers("where id=$1", server_id)
	if len(servers) > 0 {
		return &servers[0]
	}
	return nil
}
func (m *SQLManager) AddServer(name, ip, port string) {
	m.Tx.MustExec(`INSERT INTO public.server(
		id, name, ip, port)
		VALUES ($1, $2, $3, $4);`, uuid.New(), name, ip, port)
}
func (m *SQLManager) DeleteServer(id string) {
	rows := m.Tx.MustQuery("SELECT * FROM public.server_file where server_id=$1 limit 1;", id)
	if rows.Next() {
		//this server has files,forbidden deleting it
		panic("there are files on this server,you can not delete it")
	} else {
		//do deleting
		m.Tx.MustExec(`DELETE FROM public.server
		WHERE id=$1;`, id)
	}
}
func (m *SQLManager) UpdateServer(id, name, ip, port string) {
	m.Tx.MustExec(`UPDATE public.server
	SET name=$2, ip=$3, port=$4
	WHERE id=$1;`, id, name, ip, port)
}

// func (m *SQLManager) InsertFileInfo(md5, name, userId, size string, p_id *string, is_hidden bool) {
// 	query_file_info := `SELECT id
// 	FROM public.file_info where md5=$1;`
// 	file_info_id := ""
// 	file_info_row := m.Tx.MustQueryRow(query_file_info, md5)
// 	err := file_info_row.Scan(&file_info_id)
// 	if err != nil {
// 		if err != sql.ErrNoRows {
// 			panic(err)
// 		}
// 	} else {
// 		m.InsertFile(name, userId, &file_info_id, p_id, is_hidden, 1)
// 		return
// 	}
// 	file_info_id = uuid.New().String()
// 	m.Tx.MustExec("INSERT INTO public.file_info( 	id, insert_time, md5, path, user_id,size) 	VALUES ($1, $2, $3, $4, $5,$6);", file_info_id, time.Now().UTC(), md5, uuid.New(), userId, size)

// 	m.InsertFile(name, userId, &file_info_id, p_id, is_hidden, 1)

// 	add_server_file := "INSERT INTO public.server_file(id, file_info_id, insert_time, uploaded_size, is_completed, server_id)VALUES ($1,$2,$3,$4,$5,$6);"
// 	servers := m.GetServers()
// 	if len(servers) == 0 {
// 		panic("not found any server node exists")
// 	}
// 	_ = m.Tx.MustExec(add_server_file, uuid.New().String(), file_info_id, time.Now().UTC(), 0, false, servers[0].Id)
// }
func (m *SQLManager) InsertFileInfo(md5, userId string, size int64) {
	file_info_id := uuid.New().String()
	m.Tx.MustExec("INSERT INTO public.file_info( 	id, insert_time, md5, path, user_id,size) 	VALUES ($1, $2, $3, $4, $5,$6);", file_info_id, time.Now().UTC(), md5, uuid.New(), userId, size)

	add_server_file := "INSERT INTO public.server_file(id, file_info_id, insert_time, uploaded_size, is_completed, server_id)VALUES ($1,$2,$3,$4,$5,$6);"
	servers := m.GetServers()
	if len(servers) == 0 {
		panic("not found any server node exists")
	}
	m.Tx.MustExec(add_server_file, uuid.New().String(), file_info_id, time.Now().UTC(), 0, false, servers[0].Id)
}
func (m *SQLManager) DeleteFileOrDirectory(id string) {
	m.Tx.MustQuery(`call delete_file_or_directory($1);`, id)
}
func (m *SQLManager) GetFile(id string) models.File {
	files := m.getFiles("where file.id=$1", id)
	if len(files) == 1 {
		return files[0]
	} else {
		panic("not found")
	}
}
func (m *SQLManager) GetFiles(p_id, user_id string, revision int64) []models.File {
	if p_id == "" {
		if revision == -1 {
			return m.getFiles(` where is_deleted=false and p_id is null and file.user_id=$1 and "end" is null`, user_id)
		} else {
			return m.getFiles(` where is_deleted=false and p_id is null and file.user_id=$1 and start<=$2 and ("end" is null or "end">$2)`, user_id, revision)
		}
	} else {
		if revision == -1 {
			return m.getFiles(` where is_deleted=false and p_id=$1 and file.user_id=$2 and "end" is null`, p_id, user_id)
		} else {
			return m.getFiles(` where is_deleted=false and p_id=$1 and file.user_id=$2 and start<=$3 and ("end" is null or "end">$3)`, p_id, user_id, revision)
		}
	}
}
func (m *SQLManager) GetDirectories(directory_id string) []models.Directory {
	return m.getDirectories(" p_id=$1", directory_id)
}
func (m *SQLManager) getDirectories(where string, args ...interface{}) []models.Directory {
	query := `SELECT directory.id, directory.name, directory.insert_time, directory.p_id, directory.user_id,public.user.name
	FROM public.directory inner join public.user on directory.user_id=public.user.id where directory.is_deleted=false and ` + where
	rows := m.Tx.MustQuery(query, args...)
	result := []models.Directory{}
	for rows.Next() {
		model := models.Directory{}
		rows.MustScan(&model.Id, &model.Name, &model.Insert_time, &model.P_id, &model.User_id, &model.User_id)
		result = append(result, model)
	}
	return result
}
func (m *SQLManager) GetFileBlocks(server_file_id string) []models.FileBlock {
	query := `SELECT id, server_file_id, p_id, "end", start, path
    FROM public.file_block where server_file_id=$1 order by "end" desc;`
	rows := m.Tx.MustQuery(query, server_file_id)
	fileBblocks := []models.FileBlock{}
	var lastBlock *models.FileBlock
	for rows.Next() {
		fileBblock := models.FileBlock{}
		rows.MustScan(&fileBblock.Id, &fileBblock.Server_file_id, &fileBblock.P_id, &fileBblock.End, &fileBblock.Start, &fileBblock.Path)
		if lastBlock != nil && fileBblock.End != lastBlock.Start {
			continue
		}
		fileBblocks = append(fileBblocks, fileBblock)
		lastBlock = &fileBblock
	}
	return fileBblocks
}
func (m *SQLManager) getFiles(where string, args ...interface{}) []models.File {
	var sql = `SELECT file.id,file.user_id,file.start,file.is_hidden, file.insert_time, file.name,file.type,description, public.user.name,file_info.size,server_file.is_completed
	  FROM file inner join public.user on file.user_id=public.user.id
	  left join file_info on file.file_info_id=file_info.id
	  left join server_file on file_info.id=server_file.file_info_id `
	sql += where
	rows := m.Tx.MustQuery(sql, args...)
	files := []models.File{}
	for rows.Next() {
		file := &models.File{}
		rows.MustScan(&file.Id, &file.User_id, &file.Start, &file.Is_hidden, &file.InsertTime, &file.Name, &file.Type, &file.Description, &file.UserName, &file.Size, &file.Completed)
		files = append(files, *file)
	}
	return files
}

func (m *SQLManager) GetServerFileByFileId(file_id string) *models.ServerFile {
	return m.getServerFile("file.id=$1", file_id)
}
func (m *SQLManager) GetServerFileByServerFileId(server_file_id string) *models.ServerFile {
	return m.getServerFile("b.id=$1", server_file_id)
}
func (m *SQLManager) getServerFile(where string, args ...interface{}) *models.ServerFile {
	var sqlstr = `SELECT a.md5,file.id,file.name,file.is_hidden,file.p_id,a.size,a.path,b.id,file.insert_time,b.uploaded_size,b.is_completed,c.name as server_name,c.ip,c.port
	from file_info as a 
	inner join  server_file as b on a.id=b.file_info_id 
	inner join  server as c on b.server_id=c.id 
	inner join file on file.file_info_id=a.id
	where is_deleted=false and `
	sqlstr += where
	sqlstr += " order by uploaded_size desc"
	row := m.Tx.MustQueryRow(sqlstr, args...)
	data := &models.ServerFile{}
	err := row.Scan(&data.Md5, &data.File_id, &data.Name, &data.Is_hidden, &data.P_id, &data.Size, &data.Path, &data.Server_file_id, &data.Insert_time, &data.Uploaded_size, &data.Is_completed, &data.Server_name, &data.Ip, &data.Port)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		} else {
			panic(err)
		}
	}
	return data
}

func (m *SQLManager) GetServerFile(name, p_id, user_id string) *models.ServerFile {
	return m.getServerFile("file.name=$1 and file.p_id=$2 and file.user_id=$3", name, p_id, user_id)
}

func (m *SQLManager) CompleteServerFile(server_file_id string) {
	m.Tx.MustExec("update server_file set is_completed=true where id=$1", server_file_id)
}
func (m *SQLManager) AddFileBlock(server_file_id, name string, start, end int64) {
	var p_id *string = nil
	m.Tx.MustExec("INSERT INTO public.file_block(id, server_file_id, p_id, start,\"end\",path) VALUES ($1,$2,$3,$4,$5,$6);",
		uuid.New(), server_file_id, p_id, start, end, name)
	m.Tx.MustExec("update server_file set uploaded_size=$1 where id=$2 and $1>uploaded_size", end, server_file_id)
}
func (m *SQLManager) GetUserByOpId(op_id string) *models.User {
	query := `SELECT id, name FROM public."user" where op_id=$1;`
	row := m.Tx.MustQueryRow(query, op_id)
	user := new(models.User)
	err := row.Scan(&user.Id, &user.Name)
	if err != nil {
		if err != sql.ErrNoRows {
			panic(err)
		}
		return nil
	}
	return user
}
func (m *SQLManager) AddOrUpdateUser(sub string, name string) {
	user := m.GetUserByOpId(sub)
	if user == nil {
		id := uuid.New().String()
		//add user
		add := `INSERT INTO public."user"(
			id, name, insert_time,op_id)
			VALUES ($1, $2, $3,$4);`
		m.Tx.MustExec(add, id, name, time.Now().UTC(), sub)
		d := NewFileManager(m, id, 0)
		d.insertFile("", "", "", false, 2)
	} else {
		//update user name
		update := `update public."user" set name=$1 where op_id=$2`
		m.Tx.MustExec(update, name, sub)

	}
}

func (m *SQLManager) GetDirectory(path string, user_id string, revision int64) *models.Directory {
	var where string
	var args []interface{}
	p := path
	if path != "" {
		p = "/" + path
	}
	if revision == -1 {
		where = ` where "end" is null `
		args = []interface{}{user_id, p}
	} else {
		where = ` where is_deleted=false and start<=$3 and ("end" is null or "end">$3) `
		args = []interface{}{user_id, p, revision}
	}
	query := `WITH RECURSIVE CTE as(
		select id,cast (name as text) as path,is_hidden from file where p_id is null and user_id=$1 and type=2 and is_deleted=false
		UNION ALL select file.id,path || '/' || file.name,file.is_hidden from file inner join CTE on file.p_id=CTE.id` + where + `)select id,is_hidden from CTE where path=$2`

	var row = m.Tx.MustQueryRow(query, args...)
	directory := new(models.Directory)
	if err := row.Scan(&directory.Id, &directory.Is_hidden); err != nil {
		if err != sql.ErrNoRows {
			panic(err)
		} else {
			return nil
		}
	}
	return directory
}
func (m *SQLManager) SetFileHidden(file_id string, is_hidden bool) {
	m.Tx.MustExec("update file set is_hidden=$1 where id=$2", is_hidden, file_id)
}

func (m *SQLManager) GetLogs(start int64, user_id string) (logs []models.Log) {
	query := `select log.insert_time,log.file_id,log.p_id,log.action,log.number,file.name,file.type,file_info.md5,file_info.size from log inner join  file on log.file_id=file.id left join file_info on file.file_info_id=file_info.id where log.number>=$1 and log.user_id=$2 order by log.number`
	rows := m.Tx.MustQuery(query, start, user_id)
	logs = []models.Log{}
	for rows.Next() {
		log := models.Log{}
		rows.MustScan(&log.Insert_time, &log.File_id, &log.P_id, &log.Action, &log.Number, &log.File_name, &log.File_type, &log.File_md5, &log.File_size)
		logs = append(logs, log)
	}
	return logs
}

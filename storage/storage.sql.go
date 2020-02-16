package storage

import (
	"database/sql"
	"log"
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
func (m *SQLManager) Commit() {
	m.Tx.Commit()
}
func (m *SQLManager) Rollback() {
	m.Tx.Rollback()
}

func (m *SQLManager) InsertFile(name, userId, file_info_id, directory_id string, is_hidden bool) {
	insert_file := `INSERT INTO public.file(
		id, insert_time, name, description, user_id, file_info_id,is_deleted,directory_id,is_hidden)
		VALUES ($1, $2, $3, $4, $5, $6,$7,$8,$9);`
	m.Tx.MustExec(insert_file, uuid.New(), time.Now().UTC(), name, "", userId, file_info_id, false, directory_id, is_hidden)
}
func (m *SQLManager) GetServers() []models.Server {
	return m.getServers("")
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
func (m *SQLManager) InsertFileInfo(md5, name, userId, size, directory_id string, is_hidden bool) {
	query_file_info := `SELECT id
	FROM public.file_info where md5=$1;`
	file_info_id := ""
	file_info_row := m.Tx.MustQueryRow(query_file_info, md5)
	err := file_info_row.Scan(&file_info_id)
	if err != nil {
		if err != sql.ErrNoRows {
			panic(err)
		}
	} else {
		m.InsertFile(name, userId, file_info_id, directory_id, is_hidden)
		return
	}
	file_info_id = uuid.New().String()
	m.Tx.MustExec("INSERT INTO public.file_info( 	id, insert_time, md5, path, user_id,size) 	VALUES ($1, $2, $3, $4, $5,$6);", file_info_id, time.Now().UTC(), md5, uuid.New(), userId, size)

	m.InsertFile(name, userId, file_info_id, directory_id, is_hidden)

	add_server_file := "INSERT INTO public.server_file(id, file_info_id, insert_time, uploaded_size, is_completed, server_id)VALUES ($1,$2,$3,$4,$5,$6);"
	servers := m.GetServers()
	if len(servers) == 0 {
		panic("not found any server node exists")
	}
	_ = m.Tx.MustExec(add_server_file, uuid.New().String(), file_info_id, time.Now().UTC(), 0, false, servers[0].Id)
}

func (m *SQLManager) DeleteFile(id string) {
	m.Tx.MustExec("DELETE FROM public.file WHERE id=$1;", id)

}
func (m *SQLManager) GetFile(id string) models.File {
	files := m.getFiles("where id=$1", id)
	if len(files) == 1 {
		return files[0]
	} else {
		panic("not found")
	}
}
func (m *SQLManager) GetFiles(directory_id string) []models.File {
	files := m.getFiles(" where directory_id=$1", directory_id)
	return files
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
	var sql = `SELECT file.id,file.is_hidden, file.insert_time, file.name, description, public.user.name,file_info.size,server_file.is_completed
	  FROM file inner join public.user on file.user_id=public.user.id
	  inner join file_info on file.file_info_id=file_info.id
	  inner join server_file on file_info.id=server_file.file_info_id `
	sql += where
	rows := m.Tx.MustQuery(sql, args...)
	files := []models.File{}
	for rows.Next() {
		file := &models.File{}
		rows.MustScan(&file.Id, &file.Is_hidden, &file.InsertTime, &file.Name, &file.Description, &file.UserName, &file.Size, &file.Completed)
		files = append(files, *file)
	}
	return files
}

func (m *SQLManager) GetServerFileByFileId(file_id string) *models.ServerFile {
	return m.getServerFile("where file.id=$1", file_id)
}
func (m *SQLManager) GetServerFileByServerFileId(server_file_id string) *models.ServerFile {
	return m.getServerFile("where b.id=$1", server_file_id)
}
func (m *SQLManager) getServerFile(where string, args ...interface{}) *models.ServerFile {
	var sqlstr = `SELECT file.id,file.name,file.is_hidden,file.directory_id,a.size,a.path,b.id,b.insert_time,b.uploaded_size,b.is_completed,c.name as server_name,c.ip,c.port
	from file_info as a 
	inner join  server_file as b on a.id=b.file_info_id 
	inner join  server as c on b.server_id=c.id 
	inner join file on file.file_info_id=a.id `
	sqlstr += where
	sqlstr += " order by uploaded_size desc"
	row := m.Tx.MustQueryRow(sqlstr, args...)
	data := &models.ServerFile{}
	err := row.Scan(&data.File_id, &data.Name, &data.Is_hidden, &data.Directory_id, &data.Size, &data.Path, &data.Server_file_id, &data.Insert_time, &data.Uploaded_size, &data.Is_completed, &data.Server_name, &data.Ip, &data.Port)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		} else {
			panic(err)
		}
	}
	return data
}

func (m *SQLManager) GetServerFile(md5, name, directory_path, user_id string) *models.ServerFile {
	directory := m.GetDirectory(directory_path, user_id)
	if directory == nil {
		panic("parent directory not found")
	}
	return m.getServerFile("where md5=$1 and file.name=$2 and directory_id=$3", md5, name, directory.Id)
}

func (m *SQLManager) CompleteServerFile(server_file_id string) {
	m.Tx.MustExec("update server_file set is_completed=true where id=$1", server_file_id)
}
func (m *SQLManager) AddFileBlock(server_file_id, name string, start, end int64) {
	var p_id *string = nil
	m.Tx.MustExec("INSERT INTO public.file_block(id, server_file_id, p_id, start,\"end\",path) VALUES ($1,$2,$3,$4,$5,$6);",
		uuid.New(), server_file_id, p_id, start, end, name)
	server_file := m.GetServerFileByServerFileId(server_file_id)
	if end > server_file.Uploaded_size {
		m.Tx.MustExec("update server_file set uploaded_size=$1 where id=$2", end, server_file_id)
	}
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
		//create root directory for this new user
		m.AddDirectory("", "", id, false)
	} else {
		//update user name
		update := `update public."user" set name=$1 where op_id=$2`
		m.Tx.MustExec(update, name, sub)

	}
}

func (m *SQLManager) GetDirectory(path string, user_id string) *models.Directory {
	query := `WITH RECURSIVE CTE(id,p_id,name,path)as(
		select id,p_id,name,cast (name as text) as path  from directory  where is_deleted=false and user_id=$1 and name=''
		UNION ALL select directory.id,directory.p_id,directory.name,path || '/' || directory.name from CTE 
		INNER JOIN directory ON CTE.id=directory.p_id
		where $2 like path || '%'
		)select id,is_hidden from directory where id in(select id from CTE where path=$2);`
	p := path
	if path != "" {
		p = "/" + path
	}
	var row = m.Tx.MustQueryRow(query, user_id, p)
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
func (m *SQLManager) AddDirectory(path string, name string, user_id string, is_hidden bool) {
	//check if this directory already exists
	d := m.GetDirectory(strings.TrimPrefix(path+"/"+name, "/"), user_id)
	if d != nil {
		if d.Is_hidden != is_hidden {
			m.Tx.Exec("update directory set is_hidden=$1 where id=$2", is_hidden, d.Id)
		}
		return
	}
	var directory_id *string
	if name == "" && path == "" {
		directory_id = nil
	} else if name == "" && path != "" {
		panic("bad request papameter errror")
	} else {
		directory := m.GetDirectory(path, user_id)
		//this api does not allow inserting root directory
		if directory == nil {
			panic("parent directory not found")
		}
		directory_id = &directory.Id
	}
	insert := `INSERT INTO public.directory(
		id, name, insert_time,is_deleted, p_id, user_id,is_hidden)
		VALUES ($1,$2,$3,$4,$5,$6,$7);`
	m.Tx.MustExec(insert, uuid.New(), name, time.Now().UTC(), false, directory_id, user_id, is_hidden)
}
func (m *SQLManager) DeleteDirectory(id string) {
	m.Tx.MustExec("update directory set is_deleted=true where id=$1", id)
}
func (m *SQLManager) SetFileHidden(file_id string, is_hidden bool) {
	m.Tx.MustExec("update file set is_hidden=$1 where id=$2", is_hidden, file_id)
}

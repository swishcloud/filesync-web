package storage

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/swishcloud/gostudy/keygenerator"

	"github.com/google/uuid"

	_ "github.com/lib/pq"
	"github.com/swishcloud/filesync-web/storage/models"
	"github.com/swishcloud/gostudy/common"
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
func (m *SQLManager) GetPartitionLatestCommit(partition_id string) map[string]interface{} {
	query := `select * from commit where partition_id=$1 order by index desc limit 1`
	return m.Tx.ScanRow(query, partition_id)
}
func (m *SQLManager) GetPartitionFirstCommit(partition_id string) map[string]interface{} {
	query := `select * from commit where partition_id=$1 order by index limit 1`
	return m.Tx.ScanRow(query, partition_id)
}
func (m *SQLManager) GetCommits(partition_id string, from_commit string) []map[string]interface{} {
	query := `select * from commit where partition_id=$1 and index>=(select index from commit where id=$2)  order by index limit 10`
	return m.Tx.ScanRows(query, partition_id, from_commit)
}
func (m *SQLManager) GetCommitById(commit_id string) map[string]interface{} {
	query := `select * from commit where id=$1`
	return m.Tx.ScanRow(query, commit_id)
}

func (m *SQLManager) GetRecentCommits(partition_id string) []map[string]interface{} {
	query := `select * from commit where partition_id=$1 order by index desc limit 10`
	return m.Tx.ScanRows(query, partition_id)
}

func (m *SQLManager) GetCommitChanges(partition_id string, commit_id string) []map[string]interface{} {
	query := `select file.*,file_info.md5,case when end_commit_id=$2 then true else false end as del from file left join file_info on file.file_info_id=file_info.id where partition_id=$1 and (start_commit_id=$2 or end_commit_id=$2) order by del desc`
	return m.Tx.ScanRows(query, partition_id, commit_id)
}
func (m *SQLManager) GetShareByToken(token string) map[string]interface{} {
	query := `select * from public.share where token=$1`
	row := m.Tx.ScanRow(query, token)
	return row
}
func (m *SQLManager) AddShare(path string, partition_id string, commit_id string, max_commit_id string, user_id string) (token string) {
	//check if the sharing record already exists.
	query := `select * from public.share where path=$1 and commit_id=$2 and max_commit_id=$3`
	row := m.Tx.ScanRow(query, path, commit_id, max_commit_id)
	if row != nil {
		return row["token"].(string)
	}
	//create a new sharing record.
	token, err := keygenerator.NewKey(15, false, true, false, true)
	if err != nil {
		panic(err)
	}
	sql := `INSERT INTO public.share(
		id, token, path, commit_id, max_commit_id, insert_time, user_id,partition_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7,$8);`
	m.Tx.MustExec(sql, uuid.New(), token, path, commit_id, max_commit_id, time.Now().UTC(), user_id, partition_id)
	return token
}
func (m *SQLManager) GetExactFileByPath(path string, partition_id string) map[string]interface{} {
	query := `
	WITH RECURSIVE CTE AS (
		SELECT *,'' as path,0 as level from file where name='' and partition_id=$1
	  UNION ALL
		SELECT f.*,CTE.path || '/' || f.name,CTE.level+1 from file f
		inner join CTE on f.p_file_id=CTE.file_id where f.end_commit_id is null and $2 like CTE.path || '/' || f.name || '%'
	  )
	  SELECT * from CTE where path=$2 or $2='/'`
	return m.Tx.ScanRow(query, partition_id, path)
}
func (m *SQLManager) GetFileByPath(path string, partition_id string) map[string]interface{} {
	//regexp, err := regexp.Compile("/")
	//if err != nil {
	//	panic(err)
	//}
	//level := len(regexp.FindAllString(path, -1))
	query := `
WITH RECURSIVE CTE AS (
    SELECT *,'' as path,0 as level from file where name='' and partition_id=$1
  UNION ALL
    SELECT f.*,CTE.path || '/' || f.name,CTE.level+1 from file f
	inner join CTE on f.p_file_id=CTE.file_id where f.end_commit_id is null and $2 like CTE.path || '/' || f.name || '/' || '%'
  )
  SELECT * from CTE order by level desc limit 1;`
	return m.Tx.ScanRow(query, partition_id, path+"/")
}
func (m *SQLManager) SuperDoFileActions(actions []Action, user_id, partition_id string) (err error) {
	defer func() {
		err := recover()
		if err != nil {
			m.Rollback()
			panic(err)
		}
	}()
	commit_id := uuid.New().String()
	m.Tx.MustExec(`LOCK TABLE file IN SHARE ROW EXCLUSIVE MODE;`)
	m.Tx.MustExec(`INSERT INTO public.commit(
		id, partition_id, index, insert_time)
		select $1,$2, coalesce(max(index),-1)+1,$3 from commit where partition_id=$2;`, commit_id, partition_id, time.Now().UTC())
	d := NewFileManager(m, user_id)
	d.commit_id = commit_id
	d.partition_id = partition_id
	for _, action := range actions {
		err = action.Do(d)
		if err != nil {
			break
		}
	}
	if err != nil {
		m.Rollback()
	}
	return err
}

type fileManager struct {
	m            *SQLManager
	user_id      string
	commit_id    string
	partition_id string
}

func NewFileManager(m *SQLManager, user_id string) *fileManager {
	d := new(fileManager)
	d.m = m
	d.user_id = user_id
	return d
}
func (m *SQLManager) GetHistoryRevisions(path, partition_id string) []map[string]interface{} {
	query := `
	WITH RECURSIVE CTE AS (
		SELECT file.*,'' as path,commit.index as commit_index from file 
		inner join commit on file.start_commit_id=commit.id
		where p_file_id is null and commit.partition_id=$2
	  UNION ALL
		SELECT file.*,CTE.path || '/' || file.name,
		case when start_commit.index>CTE.commit_index then start_commit.index
			 else CTE.commit_index
		end
		from file 
		inner join commit start_commit on file.start_commit_id=start_commit.id
		inner join CTE on file.p_file_id=CTE.file_id
		where $1 like CTE.path || '/' || file.name || '%' and file.partition_id=$2
	  )
	SELECT CTE.*,commit.id as commit_id,commit.index as commit_index,file_info.size,public."user".name as user_name from CTE
	inner join commit on CTE.commit_index=commit.index
	left join file_info on CTE.file_info_id=file_info.id
	inner join public."user" on CTE.user_id=public."user".id
	where (CTE.path=$1 or $1='/') and commit.partition_id=$2
	order by commit.index desc;
	`
	//query = "select *,name as path from file where $1!='1xfafa' and partition_id=$2"
	rows := m.Tx.ScanRows(query, path, partition_id)
	return rows
}

//insert a file if t value is 1,if t value is 2 insert a directory.
func (d *fileManager) insertFile(name string, file_id string, p_file_id *string, md5 *string, is_hidden bool, t int, source *string) {
	if t != 1 && t != 2 {
		panic("the file type can only be 1 or 2.")
	}
	file_info_id := new(string)
	if md5 != nil {
		file_info := d.m.GetFileInfo(*md5)
		if file_info == nil {
			panic("the file does not exits for md5 " + *md5)
		}
		*file_info_id = file_info["id"].(string)
	} else {
		file_info_id = nil
	}

	if t == 1 {
		query := `select * from file where p_file_id=$1 and name=$2 and end_commit_id is null`
		file := d.m.Tx.ScanRow(query, p_file_id, name)
		if file != nil {
			if file["file_info_id"].(string) != *file_info_id || file["is_hidden"].(string) != strconv.FormatBool(is_hidden) {
				d.deleteFile(file["id"].(string))
			} else {
				//the file already exists,just return.
				return
			}
		}
	}
	insert_file := `INSERT INTO public.file(
		id, insert_time, name, description, user_id, file_info_id,is_deleted,is_hidden,type,start_commit_id,file_id,p_file_id,partition_id,source)
		VALUES ($1, $2, $3, $4, $5, $6,$7,$8,$9,$10,$11,$12,$13,$14);`
	var stored_p_file_id *string
	if p_file_id == nil || name == "" {
		if p_file_id != nil || name != "" || t != 2 {
			panic("parameters error.if add a root folder for a new user,then both p_file_id and name must be empty value.")
		}
		stored_p_file_id = nil
	} else {
		stored_p_file_id = p_file_id
	}
	d.m.Tx.MustExec(insert_file, uuid.New(), time.Now().UTC(), name, "", d.user_id, file_info_id, false, is_hidden, t, d.commit_id, file_id, stored_p_file_id, d.partition_id, source)
}

func (d *fileManager) isExists(id string) (path string, exist bool) {
	query := `
	WITH RECURSIVE CTE AS (
		SELECT *,'/'||name as p from file where id=$1 and end_commit_id is null
	  UNION ALL
		SELECT f.*,'/'||f.name||CTE.p from file f
		inner join CTE on f.file_id=CTE.p_file_id where f.end_commit_id is null
	  )
	  SELECT p_file_id,p from CTE where p_file_id is null;`
	row := d.m.Tx.ScanRow(query, id)
	if row == nil {
		return "", false
	}
	path = row["p"].(string)
	if row["p_file_id"] != "" {
		path = string([]rune(path)[1:])
	}
	return path, true
}
func (d *fileManager) deleteFile(id string) {
	d.m.Tx.MustExec(`update file set end_commit_id=$1 where id=$2 and end_commit_id is null`, d.commit_id, id)
}

func (d *fileManager) copyFile(id string, destination_path string, destination_name *string, delete_souce bool) error {
	err := validatePathFormat(destination_path)
	if err != nil {
		return err
	}
	source_path, exist := d.isExists(id)
	if !exist {
		return errors.New("this source file does not exist.")
	}

	f := d.m.GetExactFileByPath(destination_path, d.partition_id)
	if f == nil {
		return errors.New("can not find the destination path.")
	}

	source_file := d.m.GetFile(id)
	destination_file := d.m.GetFile(f["id"].(string))

	if destination_file.Type != 2 {
		return errors.New("the destination path is not a folder.")
	}

	if destination_path == source_path {
		return errors.New("the source path can not be same as the destination path.")
	}
	if source_file.Type == 2 && strings.Index(destination_path, source_path) == 0 {
		return errors.New("can not move a directory into a subdirectory of itself.")
	}
	if delete_souce {
		d.deleteFile(source_file.Id)
	}
	if destination_name == nil {
		destination_name = &source_file.Name
	}
	d.insertFile(*destination_name, source_file.File_id, &destination_file.File_id, source_file.Md5, source_file.Is_hidden, source_file.Type, &source_file.Id)
	return nil
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
func (m *SQLManager) GetFile(id string) models.File {
	files := m.getFiles(" where file.id=$1", id)
	if len(files) == 1 {
		return files[0]
	} else {
		panic("not found")
	}
}
func (m *SQLManager) getCommitById(commit_id string) map[string]interface{} {
	query := "select * from commit where id=$1"
	return m.Tx.ScanRow(query, commit_id)
}
func (m *SQLManager) getParents(path string, file_id string, commit_id string) []map[string]interface{} {
	query := `WITH RECURSIVE CTE as(
		select start_commit_id,p_file_id,cast ($1 as text) as path from file a
		where file_id=$2 and type=2 and is_deleted=false and end_commit_id is null and start_commit_id=$3
		UNION ALL 
		select file.start_commit_id,file.p_file_id,trim(trailing '/' from substring(CTE.path from '.*/')) from file
		inner join commit a on file.start_commit_id=a.id
		left join commit b on file.end_commit_id=b.id
		inner join CTE on file.file_id=CTE.p_file_id)select * from CTE 
		`
	return m.Tx.ScanRows(query, path, file_id, commit_id)
}
func (m *SQLManager) GetFiles(path string, commit_id string, max_commit_id string, partition_id string) (files []map[string]interface{}, err error) {
	histories := m.GetHistoryRevisions(path, partition_id)
	var directory map[string]interface{} = nil
	if commit_id != "" {
		for _, his := range histories {
			if his["commit_id"].(string) == commit_id {
				directory = his
				break
			}
		}
	} else if len(histories) > 0 {
		directory = histories[0]
	}
	if directory == nil {
		return nil, errors.New("the directory does not exist.")
	}
	file_id := directory["file_id"].(string)
	if max_commit_id == "" {
		max_commit_id = m.GetPartitionLatestCommit(partition_id)["id"].(string)
	}
	max_commit := m.getCommitById(max_commit_id)
	if max_commit == nil {
		return nil, errors.New("parameter error.")
	}
	max_revision, err := strconv.ParseInt(max_commit["index"].(string), 10, 64)
	directory_commit_index, err := strconv.ParseInt(directory["commit_index"].(string), 10, 64)
	if err != nil {
		return nil, err
	}
	query := `select file.*,case when start_commit.index>$3 then start_commit.id else $4 end as commit_id,file_info.size from file 
	inner join commit start_commit on file.start_commit_id=start_commit.id
	left join commit end_commit on file.end_commit_id=end_commit.id
	left join file_info on file.file_info_id=file_info.id
	where file.p_file_id=$1 and start_commit.index<=$2 and (end_commit.index is null or end_commit.index>$2) order by file.name`
	return m.Tx.ScanRows(query, file_id, max_revision, directory_commit_index, directory["commit_id"].(string)), nil
}

// func (m *SQLManager) GetFiles(p_file_id, partition_id string, revision int64) []models.File {
// 	revision = common.MaxInt64
// 	if p_file_id == "" {
// 		return m.getFiles(` where is_deleted=false and p_file_id is null and file.partition_id=$1 and start_commit.index<=$2 and (end_commit.index is null or end_commit.index>$2) order by file.name`, partition_id, revision)
// 	} else {
// 		return m.getFiles(` where is_deleted=false and p_file_id=$1 and file.partition_id=$2 and start_commit.index<=$3 and (end_commit.index is null or end_commit.index>$3) order by file.name`, p_file_id, partition_id, revision)
// 	}
// }
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
	var sql = `SELECT file.id,file.user_id,file.is_hidden, file.insert_time, file.name,file.type,file.file_id,file.p_file_id,description,file.start_commit_id, public.user.name,file_info.size,file_info.md5,server_file.is_completed
	  FROM file inner join public.user on file.user_id=public.user.id
	  left join file_info on file.file_info_id=file_info.id
	  left join server_file on file_info.id=server_file.file_info_id 
	  inner join commit start_commit on file.start_commit_id=start_commit.id
	  left join commit end_commit on file.end_commit_id=end_commit.id`
	sql += where
	rows := m.Tx.MustQuery(sql, args...)
	files := []models.File{}
	for rows.Next() {
		file := &models.File{}
		rows.MustScan(&file.Id, &file.User_id, &file.Is_hidden, &file.InsertTime, &file.Name, &file.Type, &file.File_id, &file.P_file_id, &file.Description, &file.Commit_id, &file.UserName, &file.Size, &file.Md5, &file.Completed)
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
	var sqlstr = `SELECT a.md5,file.id,file.name,file.is_hidden,file.p_file_id,a.size,a.path,b.id,file.insert_time,b.uploaded_size,b.is_completed,c.name as server_name,c.ip,c.port
	from file_info as a 
	inner join  server_file as b on a.id=b.file_info_id 
	inner join  server as c on b.server_id=c.id 
	inner join file on file.file_info_id=a.id
	where is_deleted=false and `
	sqlstr += where
	sqlstr += " order by uploaded_size desc"
	row := m.Tx.MustQueryRow(sqlstr, args...)
	data := &models.ServerFile{}
	err := row.Scan(&data.Md5, &data.File_id, &data.Name, &data.Is_hidden, &data.P_file_id, &data.Size, &data.Path, &data.Server_file_id, &data.Insert_time, &data.Uploaded_size, &data.Is_completed, &data.Server_name, &data.Ip, &data.Port)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		} else {
			panic(err)
		}
	}
	return data
}

func (m *SQLManager) GetServerFile(name, p_file_id, user_id string) *models.ServerFile {
	return m.getServerFile("file.name=$1 and file.p_file_id=$2 and file.user_id=$3", name, p_file_id, user_id)
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
	query := `SELECT id, name,partition_id FROM public."user" where op_id=$1;`
	row := m.Tx.MustQueryRow(query, op_id)
	user := new(models.User)
	err := row.Scan(&user.Id, &user.Name, &user.Partition_id)
	if err != nil {
		if err != sql.ErrNoRows {
			panic(err)
		}
		return nil
	}
	return user
}
func (m *SQLManager) addPartition() (id string) {
	id = uuid.New().String()
	m.Tx.MustExec(`INSERT INTO public.partition(
	id, insert_time)
	VALUES ($1, $2);`, id, time.Now().UTC())
	return id
}
func (m *SQLManager) AddOrUpdateUser(sub string, name string) (user *models.User, err error) {
	user = m.GetUserByOpId(sub)
	if user == nil {
		id := uuid.New().String()
		new_partition_id := m.addPartition()
		//add user
		add := `INSERT INTO public."user"(
			id, name, insert_time,op_id,partition_id)
			VALUES ($1, $2, $3,$4,$5);`
		m.Tx.MustExec(add, id, name, time.Now().UTC(), sub, new_partition_id)
		actions := []Action{}
		action := CreateDirectoryAction{Path: "/"}
		actions = append(actions, action)
		if err := m.SuperDoFileActions(actions, id, new_partition_id); err != nil {
			panic(err)
		}
		user = m.GetUserByOpId(sub)
		if user == nil {
			err = errors.New("unknown error.")
		}
	} else {
		//update user name
		update := `update public."user" set name=$1 where op_id=$2`
		m.Tx.MustExec(update, name, sub)

	}
	return user, err
}
func (m *SQLManager) GetFilePath(partition_id string, id string, revision int64) string {
	query := `WITH RECURSIVE CTE as(
		select start_commit_id,p_file_id,cast(name as text) as path from file a
		where id=$2
		UNION ALL 
		select file.start_commit_id,file.p_file_id,file.name || '/' || CTE.path from file
		inner join commit start_commit on file.start_commit_id=start_commit.id
		left join commit end_commit on file.end_commit_id=end_commit.id
		inner join CTE on file.file_id=CTE.p_file_id where file.partition_id=$1 and start_commit.index<=$3 and (end_commit.index is null or end_commit.index>$3)
)select path from CTE where p_file_id is null`
	return m.Tx.ScanRow(query, partition_id, id, revision)["path"].(string)
}
func (m *SQLManager) GetDirectory(path string, partition_id string, revision int64) *models.Directory {
	p := path
	if path != "" {
		p = "/" + path
	}
	revision = common.MaxInt64
	query := `WITH RECURSIVE CTE as(
		select id,file_id,p_file_id,cast (name as text) as path,is_hidden from file a
		where p_file_id is null and partition_id=$1 and type=2 and is_deleted=false and end_commit_id is null
		UNION ALL 
		select file.id,file.file_id,file.p_file_id,path || '/' || file.name,file.is_hidden from file
		inner join commit a on file.start_commit_id=a.id
		left join commit b on file.end_commit_id=b.id
		inner join CTE on file.p_file_id=CTE.file_id where a.index<=$3 and (b.index is null or b.index>$3) )select id,file_id,p_file_id,is_hidden from CTE where path=$2		
		`

	var row = m.Tx.MustQueryRow(query, partition_id, p, revision)
	directory := new(models.Directory)
	if err := row.Scan(&directory.Id, &directory.File_id, &directory.P_file_id, &directory.Is_hidden); err != nil {
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

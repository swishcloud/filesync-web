package models

import "time"

type User struct {
	Id           string
	Name         string
	Avatar       string
	Partition_id string
	Is_admin     bool
}
type File struct {
	Id          string
	File_id     string
	P_file_id   *string
	InsertTime  time.Time
	Description string
	Name        string
	Md5         *string
	UserName    string
	User_id     string
	Size        *int64
	Completed   *bool
	Is_hidden   bool
	Type        int
	Commit_id   string
}

type FileAction struct {
	OldPath string
	Path    string
	//action type:1 add,2 delete,3 rename
	ActionType int
	//file type:1 file,2 directory
	FileType int
	Md5      string
}
type FileBlock struct {
	Id             string
	Server_file_id string
	P_id           *string
	Start          string
	End            string
	Path           string
}

type ServerFile struct {
	Name           string
	Path           string
	Md5            string
	File_id        string
	Server_file_id string
	P_file_id      *string
	Insert_time    time.Time
	Uploaded_size  int64
	Is_completed   bool
	Server_name    string
	Ip             string
	Port           int
	Size           int64
	Is_hidden      bool
	Is_deleted     bool
}

type Server struct {
	Id   string
	Name string
	Ip   string
	Port int
}

type Directory struct {
	Id          string
	File_id     string
	P_file_id   *string
	Name        string
	Insert_time time.Time
	P_id        string
	User_id     string
	User_name   string
	Is_hidden   bool
}
type Log struct {
	Insert_time time.Time
	P_id        *string
	Action      int
	Number      int
	File_id     string
	File_type   int
	File_name   string
	File_md5    *string
	File_size   *int64
}

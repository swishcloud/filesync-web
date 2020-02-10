package models

import "time"

type User struct {
	Id     string
	Name   string
	Avatar string
}
type File struct {
	Id          string
	InsertTime  time.Time
	Description string
	Name        string
	UserName    string
	User_id     string
	Size        int64
	Completed   bool
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
	Server_file_id string
	Insert_time    time.Time
	Uploaded_size  int64
	Is_completed   bool
	Server_name    string
	Ip             string
	Port           int
	Size           int64
}

type Server struct {
	Id   string
	Name string
	Ip   string
	Port int
}

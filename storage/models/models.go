package models

import "time"

type User struct {
	Id       string
	Name     string
	Email    string
	Password string
	Avatar   *string
}
type File struct {
	Id         string
	InsertTime time.Time
	Md5        string
	Name       string
	User_id    string
}

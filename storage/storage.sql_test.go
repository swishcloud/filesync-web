package storage

import (
	"fmt"
	"log"
	"path/filepath"
	"testing"
)

var db_conn_info string = "postgres://filesync:secret@192.168.1.1:6010/filesync?sslmode=disable"
var user_id string = "40aee33a-7700-4537-ad0c-7ad6e61ec326"
var root_directory_id string = "5acb3705-68de-4391-8c10-488f348877b0"

//test Add,Delete,Rename,Move,Query.
func Test1(t *testing.T) {
	manager := NewSQLManager(db_conn_info)

	test_md5 := "test md5 string"
	manager.InsertFileInfo(test_md5, user_id, 0)

	a := []Action{}
	a = append(a, CreateDirectoryAction{Path: "/home1"})
	a = append(a, CreateDirectoryAction{Path: "/home1/sub1"})
	a = append(a, CreateDirectoryAction{Path: "/home1/sub1/subsub1"})
	a = append(a, CreateFileAction{Location: "/home1/sub1/subsub1", Name: "file1", Md5: test_md5})
	a = append(a, CreateFileAction{Location: "/home1/sub1/subsub1", Name: "file2", Md5: test_md5})
	a = append(a, CreateDirectoryAction{Path: "/home2"})
	a = append(a, CreateDirectoryAction{Path: "/home2/sub2"})
	a = append(a, CreateDirectoryAction{Path: "/home3"})

	a = append(a, CreateDirectoryAction{Path: "/home1"})
	a = append(a, CreateDirectoryAction{Path: "/home1/sub1"})
	a = append(a, CreateDirectoryAction{Path: "/home1/sub1/subsub1"})
	a = append(a, CreateDirectoryAction{Path: "/home2"})
	a = append(a, CreateDirectoryAction{Path: "/home2/sub2"})
	a = append(a, CreateDirectoryAction{Path: "/home3"})
	err := manager.SuperDoFileActions(a, user_id)
	if err != nil {
		t.Fatal(err)
	}

	//print files
	paths := []string{}
	getFiles(manager, "/", root_directory_id, user_id, -1, &paths)
	printPaths(paths)

	file := manager.GetExactFileByPath("/home1/sub1/subsub1", user_id)
	if file == nil {
		log.Fatal("can not find specified path.")
	}

	//Rename
	a = []Action{}
	a = append(a, RenameAction{Id: file["id"].(string), NewName: "renamed"})
	err = manager.SuperDoFileActions(a, user_id)
	if err != nil {
		t.Fatal(err)
	}

	//print files
	paths = []string{}
	getFiles(manager, "/", root_directory_id, user_id, -1, &paths)
	printPaths(paths)

	file = manager.GetExactFileByPath("/home3", user_id)
	if file == nil {
		log.Fatal("can not find specified path.")
	}
	//delete
	a = []Action{}
	a = append(a, DeleteAction{Id: file["id"].(string)})
	err = manager.SuperDoFileActions(a, user_id)
	if err != nil {
		log.Fatal(err)
	}

	//print files
	paths = []string{}
	getFiles(manager, "/", root_directory_id, user_id, -1, &paths)
	printPaths(paths)

	file = manager.GetExactFileByPath("/home1/sub1/renamed", user_id)
	if file == nil {
		log.Fatal("can not find specified path.")
	}

	//Move
	a = []Action{}
	a = append(a, MoveAction{Id: file["id"].(string), DestinationPath: "/"})
	err = manager.SuperDoFileActions(a, user_id)
	if err != nil {
		log.Fatal(err)
	}

	//print files
	paths = []string{}
	getFiles(manager, "/", root_directory_id, user_id, -1, &paths)
	printPaths(paths)
}
func printPaths(paths []string) {
	fmt.Println("file list:")
	for _, p := range paths {
		fmt.Println(p)
	}
}
func getFiles(manager *SQLManager, directory_path, p_file_id, user_id string, revision int64, paths *[]string) {
	files := manager.GetFiles(p_file_id, user_id, -1)
	for _, f := range files {
		full_path := filepath.Join(directory_path, f.Name)
		*paths = append(*paths, full_path)
		if f.Type == 2 {
			getFiles(manager, full_path, f.File_id, user_id, revision, paths)
		}
	}
}

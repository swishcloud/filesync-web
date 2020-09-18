package storage

import (
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/swishcloud/gostudy/common"
)

var db_conn_info string = "postgres://filesync:secret@192.168.1.1:6010/filesync?sslmode=disable"
var user_id string
var partition_id string
var root_directory_id string
var test_md5 string
var manager Storage

func init() {
	manager = NewSQLManager(db_conn_info)
	test_md5 = "test md5 string"
	user, err := manager.AddOrUpdateUser("841e98ee-5eeb-4c65-ad57-954452912fe9", "unit test")
	if err != nil {
		panic(err)
	}
	user_id = user.Id
	partition_id = user.Partition_id
	if manager.GetFileInfo(test_md5) == nil {
		manager.InsertFileInfo(test_md5, user_id, 0)
	}
	root_file := manager.GetExactFileByPath("/", partition_id)
	root_directory_id = root_file["file_id"].(string)
	err = manager.Commit()
	if err != nil {
		panic(err)
	}
}
func TestATonsOfFiles(t *testing.T) {
	i := 0
	for {
		if i == 1000000 {
			break
		}
		manager = NewSQLManager(db_conn_info)
		a := []Action{}
		a_f := CreateFileAction{Location: "/ATonsOfFiles", Name: "file_a" + strconv.Itoa(i), Md5: test_md5}
		a = append(a, a_f)
		log.Println("add "+strconv.Itoa(i+1)+"th file:", a_f.Location+"/"+a_f.Name)
		err := manager.SuperDoFileActions(a, user_id, partition_id)
		if err != nil {
			t.Fatal(err)
		}
		err = manager.Commit()
		if err != nil {
			panic(err)
		}
		i++
	}
}
func TestHistories(t *testing.T) {
	a := []Action{}
	a = append(a, CreateDirectoryAction{Path: "/folder"})
	a = append(a, CreateFileAction{Location: "/folder", Name: "file", Md5: test_md5})
	err := manager.SuperDoFileActions(a, user_id, partition_id)
	if err != nil {
		t.Fatal(err)
	}
	//print files
	paths := []string{}
	getFiles(manager, "/", partition_id, -1, &paths)
	printPaths(paths)

	file := manager.GetExactFileByPath("/folder", partition_id)
	if file == nil {
		log.Fatal("can not find specified path.")
	}
	a = []Action{}
	a = append(a, CreateDirectoryAction{Path: "/folder"})
	a = append(a, RenameAction{Id: file["id"].(string), NewName: "folder(renamed)"})
	err = manager.SuperDoFileActions(a, user_id, partition_id)
	if err != nil {
		t.Fatal(err)
	}
	//print files
	paths = []string{}
	getFiles(manager, "/", partition_id, -1, &paths)
	printPaths(paths)

	file = manager.GetExactFileByPath("/folder(renamed)", partition_id)
	if file == nil {
		log.Fatal("can not find specified path.")
	}
	a = []Action{}
	a = append(a, RenameAction{Id: file["id"].(string), NewName: "folder"})
	err = manager.SuperDoFileActions(a, user_id, partition_id)
	if err != nil {
		t.Fatal(err)
	}
	//print files
	paths = []string{}
	getFiles(manager, "/", partition_id, -1, &paths)
	printPaths(paths)

	revisions := manager.GetHistoryRevisions("/folder/file", partition_id)
	printRevision(revisions)
	revisions = manager.GetHistoryRevisions("/folder(renamed)/file", partition_id)
	printRevision(revisions)
}

func printRevision(revisions []map[string]interface{}) {
	for _, r := range revisions {
		fmt.Println(r["id"].(string), r["file_id"].(string), r["p_file_id"], r["commit_index"].(string), r["path"].(string))
	}
}

//test Add,Delete,Rename,Move,Query.
func TestCURD(t *testing.T) {
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
	err := manager.SuperDoFileActions(a, user_id, partition_id)
	if err != nil {
		t.Fatal(err)
	}

	//print files
	paths := []string{}
	getFiles(manager, "/", partition_id, -1, &paths)
	printPaths(paths)

	file := manager.GetExactFileByPath("/home1/sub1/subsub1", partition_id)
	if file == nil {
		log.Fatal("can not find specified path.")
	}

	//Rename
	a = []Action{}
	a = append(a, RenameAction{Id: file["id"].(string), NewName: "renamed"})
	err = manager.SuperDoFileActions(a, user_id, partition_id)
	if err != nil {
		t.Fatal(err)
	}

	//print files
	paths = []string{}
	getFiles(manager, "/", partition_id, -1, &paths)
	printPaths(paths)

	file = manager.GetExactFileByPath("/home3", partition_id)
	if file == nil {
		log.Fatal("can not find specified path.")
	}
	//delete
	a = []Action{}
	a = append(a, DeleteAction{Id: file["id"].(string)})
	err = manager.SuperDoFileActions(a, user_id, partition_id)
	if err != nil {
		log.Fatal(err)
	}

	//print files
	paths = []string{}
	getFiles(manager, "/", partition_id, -1, &paths)
	printPaths(paths)

	file = manager.GetExactFileByPath("/home1/sub1/renamed", partition_id)
	if file == nil {
		log.Fatal("can not find specified path.")
	}

	//Move
	a = []Action{}
	a = append(a, MoveAction{Id: file["id"].(string), DestinationPath: "/"})
	err = manager.SuperDoFileActions(a, user_id, partition_id)
	if err != nil {
		log.Fatal(err)
	}

	//print files
	paths = []string{}
	getFiles(manager, "/", partition_id, -1, &paths)
	printPaths(paths)
}
func printPaths(paths []string) {
	fmt.Println("file list:")
	for _, p := range paths {
		fmt.Println(p)
	}
}
func getFiles(manager Storage, directory_path, partition_id string, revision int64, paths *[]string) {
	files, err := manager.GetFiles(directory_path, nil, common.MaxInt64, partition_id)
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		full_path := filepath.Join(directory_path, f["name"].(string))
		*paths = append(*paths, full_path)
		if f["type"].(string) == "2" {
			getFiles(manager, full_path, partition_id, revision, paths)
		}
	}
}

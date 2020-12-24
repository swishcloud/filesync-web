package storage

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/google/uuid"
)

type Action interface {
	Do(m *fileManager) error
}
type CreateDirectoryAction struct {
	Path     string
	IsHidden bool
}

func (action CreateDirectoryAction) Do(m *fileManager) error {
	path := action.Path
	err := validatePathFormat(path)
	if err != nil {
		return err
	}
	for {
		file := m.m.GetFileByPath(path, m.partition_id)
		if file == nil {
			//create root directory
			if path != "/" {
				return errors.New("can not find root directory at all")
			}
			m.insertFile("", uuid.New().String(), nil, nil, false, 2, nil)
			return nil
		}
		p := file["path"].(string)
		loop_file_id := file["file_id"].(string)
		if p != path {
			name := string([]rune(path)[len([]rune(p)):])
			regexp, err := regexp.Compile("[^/]+")
			if err != nil {
				panic(err)
			}
			name = regexp.FindString(name)
			fmt.Println("found parent directory " + p + " for " + path + ",creating directory " + name + " under it.")
			m.insertFile(name, uuid.New().String(), &loop_file_id, nil, action.IsHidden, 2, nil)
		} else {
			if file["type"].(string) != "2" {
				return errors.New("There is already a file with the same path as you specified:" + path)
			}
			return nil
		}
	}
}

type CreateFileAction struct {
	Name     string
	Md5      string
	Location string
	IsHidden bool
}

func (action CreateFileAction) Do(m *fileManager) error {
	err := validatePathFormat(action.Location)
	if err != nil {
		return err
	}
	err = validatePathFormat("/" + action.Name)
	if err != nil {
		return err
	}
	if action.Name == "" || action.Md5 == "" {
		return errors.New("not set name or md5 value.")
	}
	location := m.m.GetExactFileByPath(action.Location, m.partition_id)
	if location == nil {
		return errors.New("can not find the destination path.")
	}
	p_file_id := location["file_id"].(string)
	m.insertFile(action.Name, uuid.New().String(), &p_file_id, &action.Md5, action.IsHidden, 1, nil)
	return nil
}

type DeleteAction struct {
	Id string
}

func (action DeleteAction) Do(m *fileManager) error {
	if _, exist := m.isExists(action.Id); !exist {
		return errors.New("this source file does not exist.")
	}
	m.deleteFile(action.Id)
	return nil
}

//

type DeleteByPathAction struct {
	Path      string
	Commit_id string
	File_type int
}

func (action DeleteByPathAction) Do(m *fileManager) error {
	file := m.m.GetFile(action.Path, m.partition_id, action.Commit_id, action.File_type)
	if file == nil {
		return errors.New("this source file does not exist.")
	}
	m.deleteFile(file["id"].(string))
	return nil
}

//

type RenameAction struct {
	Id      string
	NewName string
}

func (action RenameAction) Do(m *fileManager) error {
	index, err := strconv.ParseInt(m.m.getCommitById(m.commit_id)["index"].(string), 10, 64)
	if err != nil {
		return err
	}
	path, err := m.m.GetFilePath(m.partition_id, action.Id, index-1)
	if err != nil {
		return err
	}
	return m.copyFile(action.Id, filepath.Dir(path), &action.NewName, true)
}

type CopyAction struct {
	Id              string
	DestinationPath string
}

func (action CopyAction) Do(m *fileManager) error {
	return m.copyFile(action.Id, action.DestinationPath, nil, false)
}

type MoveAction struct {
	Id              string
	DestinationPath string
}

func (action MoveAction) Do(m *fileManager) error {
	return m.copyFile(action.Id, action.DestinationPath, nil, true)
}

func validatePathFormat(path string) error {
	regexp := regexp.MustCompile(`^\/([^\\\/:*?"<>|]+(\/[^\\\/:*?"<>|]+)*){0,1}$`)
	if regexp.MatchString(path) {
		return nil
	} else {
		return errors.New("path format error.")
	}
}

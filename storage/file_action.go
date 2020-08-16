package storage

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

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
		file := m.m.GetFileByPath(path, m.user_id)
		if file == nil {
			//create root directory
			if path != "/" {
				return errors.New("can not find root directory at all")
			}
			m.insertFile("", uuid.New().String(), nil, nil, false, 2)
			return nil
		}
		p := file["path"].(string)
		loop_file_id := file["file_id"].(string)
		if p != path {
			name := string([]rune(path)[len(p):])
			regexp, err := regexp.Compile("[^/]+")
			if err != nil {
				panic(err)
			}
			name = regexp.FindString(name)
			fmt.Println("found parent directory " + p + " for " + path + ",creating directory " + name + " under it.")
			m.insertFile(name, uuid.New().String(), &loop_file_id, nil, action.IsHidden, 2)
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
	location := m.m.GetExactFileByPath(action.Location, m.user_id)
	if location == nil {
		return errors.New("can not find the destination path.")
	}
	p_file_id := location["file_id"].(string)
	m.insertFile(action.Name, uuid.New().String(), &p_file_id, &action.Md5, action.IsHidden, 1)
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

type RenameAction struct {
	Id      string
	NewName string
}

func (action RenameAction) Do(m *fileManager) error {
	err := validatePathFormat("/" + action.NewName)
	if err != nil {
		return err
	}
	if _, exist := m.isExists(action.Id); !exist {
		return errors.New("this source file does not exist.")
	}
	m.deleteFile(action.Id)
	file := m.m.GetFile(action.Id)
	m.insertFile(action.NewName, file.File_id, file.P_file_id, file.Md5, file.Is_hidden, file.Type)
	return nil
}

type MoveAction struct {
	Id              string
	DestinationPath string
}

func (action MoveAction) Do(m *fileManager) error {
	err := validatePathFormat(action.DestinationPath)
	if err != nil {
		return err
	}
	source_path, exist := m.isExists(action.Id)
	if !exist {
		return errors.New("this source file does not exist.")
	}

	f := m.m.GetExactFileByPath(action.DestinationPath, m.user_id)
	if f == nil {
		return errors.New("can not find the destination path.")
	}

	source_file := m.m.GetFile(action.Id)
	destination_file := m.m.GetFile(f["id"].(string))

	if destination_file.Type != 2 {
		return errors.New("the destination path is not a folder.")
	}

	if action.DestinationPath == source_path {
		return errors.New("the source path can not be same as the destination path.")
	}
	if source_file.Type == 2 && strings.Index(action.DestinationPath, source_path) == 0 {
		return errors.New("can not move a directory into a subdirectory of itself.")
	}

	m.deleteFile(source_file.Id)
	m.insertFile(source_file.Name, source_file.File_id, &destination_file.File_id, source_file.Md5, source_file.Is_hidden, source_file.Type)
	return nil
}

func validatePathFormat(path string) error {
	regexp := regexp.MustCompile(`^\/([^\\\/:*?"<>|]+(\/[^\\\/:*?"<>|]+)*){0,1}$`)
	if regexp.MatchString(path) {
		return nil
	} else {
		return errors.New("path format error.")
	}
}

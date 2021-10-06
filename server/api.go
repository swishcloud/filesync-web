package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"github.com/swishcloud/filesync-web/storage"
	"github.com/swishcloud/goweb"
	"github.com/swishcloud/goweb/auth"
	"golang.org/x/oauth2"
)

const (
	API_PATH_File_INFO      = "/api/file-info"
	API_PATH_File_Upload    = "/api/file_upload"
	API_PATH_File           = "/api/file"
	API_PATH_File_Block     = "/api/file-block"
	API_PATH_Login          = "/api/login"
	API_PATH_Exchange_Token = "/api/exchange_token"
	API_PATH_Auth_Code_Url  = "/api/auth_code_url"
	API_PATH_Directory      = "/api/directory"
	API_PATH_Log            = "/api/log"
	API_PATH_Commit_Changes = "/api/commit/changes"
	API_PATH_Files          = "/api/files"
	API_Reset_Server_File   = "/api/reset-server-file"
)

func (s *FileSyncWebServer) bindApiHandlers(group *goweb.RouterGroup) {
	group.Use(func(ctx *goweb.Context) { ctx.Writer.EnsureInitialzed(false) })
	group.Use(s.apiMiddleware())
	group.GET(API_PATH_File_INFO, s.fileInfoApiGetHandler())
	group.POST(API_PATH_File_Upload, s.fileUploadApiPostHandler())
	group.GET(API_PATH_File, s.fileApiGetHandler())
	group.POST(API_PATH_File, s.fileApiPostHandler())
	group.PUT(API_PATH_File, s.fileApiPutHandler())
	group.DELETE(API_PATH_File, s.fileApiDeleteHandler())
	group.GET(API_PATH_File_Block, s.fileBlockApiGetHandler())
	group.POST(API_PATH_File_Block, s.fileBlockApiPostHandler())
	group.POST(API_PATH_Login, s.fileBlockApiPostHandler())
	group.POST(API_PATH_Exchange_Token, s.exchangeTokenApiPostHandler())
	group.GET(API_PATH_Auth_Code_Url, s.authCodeURLApiGetHandler())
	group.POST(API_PATH_Directory, s.directoryApiPostHandler())
	group.GET(API_PATH_Directory, s.directoryApiGetHandler())
	group.GET(API_PATH_Log, s.logApiGetHandler())
	group.GET(API_PATH_Commit_Changes, s.commitChangesGetHandler())
	group.GET(API_PATH_Files, s.filesGetHandler())
	group.POST(API_Reset_Server_File, s.resetServerFile())
}

func (s *FileSyncWebServer) apiMiddleware() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.Writer.EnsureInitialzed(true)
		if tokenstr, err := auth.GetBearerToken(ctx); err == nil {
			token := &oauth2.Token{AccessToken: tokenstr}
			if ok, sub, err := auth.CheckToken(s.rac, token, s.config.OAuth.IntrospectTokenURL, s.skip_tls_verify); err == nil {
				if !ok {
					panic("the session has expired.")
				}
				user := s.GetStorage(ctx).GetUserByOpId(sub)
				if user == nil {
					s.addOrUpdateUser(ctx, token)
					user = s.GetStorage(ctx).GetUserByOpId(sub)
				}
				ctx.Data["user"] = user
			} else {
				panic(err)
			}
		} else {
			panic(err)
		}
	}
}
func (s *FileSyncWebServer) fileInfoApiGetHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		md5 := ctx.Request.FormValue("md5")
		size, err := strconv.ParseInt(ctx.Request.FormValue("size"), 10, 64)
		if err != nil {
			panic(err)
		}
		fileInfo := s.GetStorage(ctx).GetFileInfo(md5)
		if fileInfo == nil {
			s.GetStorage(ctx).InsertFileInfo(md5, s.MustGetLoginUser(ctx).Id, size)
			fileInfo = s.GetStorage(ctx).GetFileInfo(md5)
		}
		ctx.Success(fileInfo)
	}
}
func (s *FileSyncWebServer) directoryApiGetHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		// path := ctx.Request.FormValue("path")
		// revision, err := strconv.ParseInt(ctx.Request.FormValue("r"), 10, 64)
		// if err != nil {
		// 	panic(err)
		// }
		// ctx.Success(s.GetStorage(ctx).GetDirectory(path, s.MustGetLoginUser(ctx).Id, revision))
	}
}
func (s *FileSyncWebServer) directoryApiPostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {

		ctx.Success(nil)
	}
}

func (s *FileSyncWebServer) fileUploadApiPostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		// file, fileHeader, err := ctx.Request.FormFile("file")
		// if err != nil {
		// 	panic(err)
		// }
		// uuid := uuid.New().String()
		// path := s.config.upload_folder + uuid + filepath.Ext(fileHeader.Filename)
		// out, err := os.Create(path)
		// defer out.Close()
		// if err != nil {
		// 	panic(err)
		// }
		// io.Copy(out, file)
		// s.GetStorage(ctx).InsertFileInfo(uuid, fileHeader.Filename, uuid, "file_size", nil, false)
		ctx.Writer.WriteHeader(204)
	}
}

func (s *FileSyncWebServer) fileApiGetHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		path := ctx.Request.FormValue("path")
		commit_id := ctx.Request.FormValue("commit_id")
		user := s.MustGetLoginUser(ctx)
		file := s.GetStorage(ctx).GetFile(path, user.Partition_id, commit_id, 1)
		if file == nil {
			panic("not found the file")
		}
		server_file := s.GetStorage(ctx).GetServerFileByFileId(file["id"].(string))
		ctx.Success(server_file)
	}
}

func (s *FileSyncWebServer) fileApiPostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		err := ctx.Request.ParseMultipartForm(ctx.Request.ContentLength)
		if err != nil {
			panic(err)
		}
		directory_actions_json := ctx.Request.PostForm.Get("directory_actions")
		file_actions_json := ctx.Request.PostForm.Get("file_actions")
		delete_actions_json := ctx.Request.PostForm.Get("delete_actions")
		delete_by_path_actions_json := ctx.Request.PostForm.Get("delete_by_path_actions")
		directory_actions := []storage.CreateDirectoryAction{}
		file_actions := []storage.CreateFileAction{}
		delete_actions := []storage.DeleteAction{}
		delete_by_path_actions := []storage.DeleteByPathAction{}
		err = json.Unmarshal([]byte(directory_actions_json), &directory_actions)
		if err != nil {
			fmt.Println("failed to parse the json:" + directory_actions_json)
			panic(err)
		}
		json.Unmarshal([]byte(file_actions_json), &file_actions)
		if err != nil {
			panic(err)
		}
		json.Unmarshal([]byte(delete_actions_json), &delete_actions)
		if err != nil {
			panic(err)
		}
		json.Unmarshal([]byte(delete_by_path_actions_json), &delete_by_path_actions)
		if err != nil {
			panic(err)
		}
		actions := []storage.Action{}
		for _, a := range directory_actions {
			actions = append(actions, a)
		}
		for _, a := range file_actions {
			actions = append(actions, a)
		}
		for _, a := range delete_actions {
			actions = append(actions, a)
		}
		for _, a := range delete_by_path_actions {
			actions = append(actions, a)
		}
		if len(actions) > 0 {
			if err := s.GetStorage(ctx).SuperDoFileActions(actions, s.MustGetLoginUser(ctx).Id, s.MustGetLoginUser(ctx).Partition_id); err != nil {
				panic(err)
			}
		}
		ctx.Success(nil)
	}
}

func (s *FileSyncWebServer) fileApiPutHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		server_file_id := ctx.Request.FormValue("server_file_id")
		s.GetStorage(ctx).CompleteServerFile(server_file_id)
		ctx.Success(nil)
	}
}
func (s *FileSyncWebServer) fileApiDeleteHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		//id := ctx.Request.FormValue("file_id")
		//s.GetStorage(ctx).DeleteFileOrDirectory(id)
		ctx.Success(nil)
	}
}
func (s *FileSyncWebServer) fileBlockApiGetHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		server_file_id := ctx.Request.FormValue("server_file_id")
		data := s.GetStorage(ctx).GetFileBlocks(server_file_id)
		ctx.Success(data)
	}
}
func (s *FileSyncWebServer) fileBlockApiPostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		server_file_id := ctx.Request.FormValue("server_file_id")
		start := ctx.Request.FormValue("start")
		end := ctx.Request.FormValue("end")
		name := ctx.Request.FormValue("name")
		start_i, err := strconv.ParseInt(start, 10, 64)
		if err != nil {
			panic(err)
		}
		end_i, err := strconv.ParseInt(end, 10, 64)
		if err != nil {
			panic(err)
		}
		s.GetStorage(ctx).AddFileBlock(server_file_id, name, start_i, end_i)
	}
}

func (s *FileSyncWebServer) loginApiPostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		state := ctx.Request.FormValue("state")
		s.oAuth2Config.RedirectURL = s.config.OAuth.NativeAppRedirectURL
		url := s.oAuth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
		ctx.Success(url)
	}
}

func (s *FileSyncWebServer) exchangeTokenApiPostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		code := ctx.Request.FormValue("code")
		s.oAuth2Config.RedirectURL = s.config.OAuth.NativeAppRedirectURL
		token, err := s.oAuth2Config.Exchange(context.WithValue(context.Background(), "", s.httpClient), code)
		if err != nil {
			panic(err)
		}
		ctx.Success(token)
	}
}

func (s *FileSyncWebServer) authCodeURLApiGetHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		state := ctx.Request.FormValue("state")
		s.oAuth2Config.RedirectURL = s.config.OAuth.NativeAppRedirectURL
		url := s.oAuth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
		ctx.Success(url)
	}
}

func (s *FileSyncWebServer) logApiGetHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		/* start_number, err := strconv.ParseInt(ctx.Request.FormValue("start"), 10, 64)
		if err != nil {
			panic(err)
		}
		log := s.GetStorage(ctx).GetLogs(start_number, s.MustGetLoginUser(ctx).Id)
		ctx.Success(log) */
	}
}
func (s *FileSyncWebServer) commitChangesGetHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		commit_id := ctx.Request.FormValue("commit_id")
		partition_id := s.MustGetLoginUser(ctx).Partition_id
		next_commit := s.GetStorage(ctx).GetNextCommit(partition_id, commit_id)
		if next_commit == nil {
			ctx.Success(nil)
			return
		}
		file_changes := s.GetCommitFileChanges(s.GetStorage(ctx), next_commit["id"].(string), partition_id)
		result := struct {
			Commit_id string
			Changes   []FileChange
		}{Commit_id: next_commit["id"].(string), Changes: file_changes}
		ctx.Success(result)
	}
}
func (s *FileSyncWebServer) filesGetHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		path := ctx.Request.FormValue("path")
		commit_id := ctx.Request.FormValue("commit_id")
		max_commit_id := ctx.Request.FormValue("max")
		if _, err := uuid.Parse(max_commit_id); err != nil {
			panic(err)
		}
		user := s.MustGetLoginUser(ctx)
		if commit_id == "" {
			commit := s.GetStorage(ctx).GetPartitionFirstCommit(user.Partition_id)
			commit_id = commit["id"].(string)
		}
		files, err := s.GetStorage(ctx).GetFiles(path, commit_id, max_commit_id, user.Partition_id)
		if err != nil {
			panic(err)
		}
		ctx.Success(files)
	}
}
func (s *FileSyncWebServer) resetServerFile() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.FormValue("id")
		user := s.MustGetLoginUser(ctx)
		s.GetStorage(ctx).ResetServerFile(user.Partition_id, id)
		ctx.Success(nil)
	}
}

package server

import (
	"context"
	"fmt"
	"image/png"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/swishcloud/filesync-web/storage"
	"github.com/swishcloud/filesync-web/storage/models"
	"github.com/swishcloud/filesync/message"
	"github.com/swishcloud/filesync/session"
	"github.com/swishcloud/gostudy/common"
	"github.com/swishcloud/goweb"
	"github.com/swishcloud/goweb/auth"
	"golang.org/x/oauth2"
)

type pageModel struct {
	Data             interface{}
	MobileCompatible bool
	User             *models.User
	WebsiteName      string
	PageTitle        string
}

func (s *FileSyncWebServer) MustGetLoginUser(ctx *goweb.Context) *models.User {
	user, err := s.GetLoginUser(ctx)
	if err != nil {
		panic(err)
	}
	return user
}
func (s *FileSyncWebServer) newPageModel(ctx *goweb.Context, data interface{}) pageModel {
	m := pageModel{}
	m.Data = data
	m.MobileCompatible = true
	if ctx.Data["user"] != nil {
		m.User = ctx.Data["user"].(*models.User)
	}
	m.WebsiteName = "filesync-web"
	return m
}

const (
	Path_Index              = "/"
	Path_File               = "/file"
	Path_File_Rename        = "/file_rename"
	Path_File_Edit          = "/file_edit"
	Path_File_History       = "/file/history"
	Path_File_Move          = "/file_move"
	Path_File_Copy          = "/file_copy"
	Path_File_List          = "/file/list"
	Path_Login              = "/login"
	Path_Login_Callback     = "/login-callback"
	Path_Logout             = "/logout"
	Path_Download_File      = "/file/download"
	Path_Server             = "/server"
	Path_Server_Edit        = "/server_edit"
	Path_Directory          = "/directory"
	Path_File_Upload        = "/file/upload"
	Path_File_Share         = "/file/share"
	Path_File_Commit        = "/file/commit"
	Path_File_Commit_Detail = "/file/commit/detail"
	Path_QRCode             = "/qr_code"
)

func (s *FileSyncWebServer) bindHandlers(root *goweb.RouterGroup) {
	open := root.Group()
	root.Use(s.genericMiddleware())
	open.RegexMatch(regexp.MustCompile(`/static/.+`), func(context *goweb.Context) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))).ServeHTTP(context.Writer, context.Request)
	})
	open.RegexMatch(regexp.MustCompile(`/sh/.+`), func(ctx *goweb.Context) {
		r := regexp.MustCompile("[^/]+")
		strs := r.FindAllString(ctx.Request.URL.Path, -1)
		token := strs[1]
		relative_path := strings.Join(strs[2:], "/")
		share := s.GetStorage(ctx).GetShareByToken(token)
		path := filepath.Join(share["path"].(string), relative_path)
		share_partition_id := share["partition_id"].(string)
		dl := ctx.Request.FormValue("dl")
		share_max_commit_id := share["max_commit_id"].(string)
		max_commit := s.GetStorage(ctx).GetCommitById(share_max_commit_id)
		if max_commit == nil {
			panic("parameter error.")
		}
		share_max_revision, err := strconv.ParseInt(max_commit["index"].(string), 10, 64)
		if err != nil {
			panic(err)
		}
		file := s.GetStorage(ctx).GetHistoryRevisions(path, share_partition_id, share_max_revision)[0]
		file_identifier := file["id"].(string)
		server_file := s.GetStorage(ctx).GetServerFileByFileId(file_identifier)
		typed_file := s.GetStorage(ctx).GetFile(file_identifier)
		if file == nil {
			panic("not found")
		}
		if dl == "1" { //directly download
			if file["type"].(string) == "2" {
			} else { //it's a file
				server_file := s.GetStorage(ctx).GetServerFileByFileId(file["id"].(string))
				conn, err := net.Dial("tcp", server_file.Ip+":"+strconv.Itoa(server_file.Port))
				if err != nil {
					panic(err)
				}
				s := session.NewSession(conn)
				msg := message.NewMessage(message.MT_Download_File)
				msg.Header["path"] = server_file.Path
				_, err = s.Fetch(msg, nil)
				if err != nil {
					panic(err)
				}
				ctx.Writer.Header().Set("Content-Type", "application/octet-stream")
				ctx.Writer.Header().Set("Content-Disposition", `attachment; filename="`+server_file.Name+`"`)
				ctx.Writer.Header().Set("Content-Length", strconv.FormatInt(server_file.Size, 10))
				if ctx.Writer.Compress {
					panic("compression should not pick up")
				}
				_, err = io.CopyN(ctx.Writer, s, server_file.Size)
				if err != nil {
					log.Println(err)
					panic(err)
				}
				s.Close()

			}
		} else {
			if file["type"].(string) == "2" { //it's a directory
				files, err := s.GetStorage(ctx).GetFiles(path, file["commit_id"].(string), share_max_commit_id, share_partition_id)
				if err != nil {
					panic(err)
				}
				data := struct {
					Path  string
					Files []map[string]interface{}
				}{Files: files}
				data.Path = filepath.Base(share["path"].(string))
				model := s.newPageModel(ctx, data)
				model.PageTitle = data.Path
				ctx.FuncMap["detailUrl"] = func(file map[string]interface{}) (string, error) {
					if file["type"] == "1" {
						return s.generateShareUrl(filepath.Join("/", relative_path, "/", file["name"].(string)), token, "0"), nil
					} else {
						return s.generateShareUrl(filepath.Join("/", relative_path, "/", file["name"].(string)), token, "0"), nil

					}
				}
				ctx.FuncMap["isHidden"] = func(isHidden bool) (string, error) {
					if isHidden {
						return "true", nil
					} else {
						return "false", nil
					}
				}
				ctx.RenderPage(model, "templates/layout.html", "templates/share_file_list.html")
			} else {
				data := struct {
					DownloadUrl string
					QRCodeUrl   string
					Path        string
					File        models.File
					ServerFile  models.ServerFile
					History     map[string]interface{}
				}{Path: path, File: typed_file, ServerFile: *server_file, History: file}
				data.DownloadUrl = "https://" + s.config.Website_domain + s.generateShareUrl(filepath.Join("/", relative_path), token, "1")
				parameters := url.Values{}
				parameters.Add("str", "https://"+s.config.Website_domain+s.generateShareUrl(filepath.Join("/", relative_path), token, "0"))
				data.QRCodeUrl = Path_QRCode + "?" + parameters.Encode()
				model := s.newPageModel(ctx, data)
				model.PageTitle = data.Path
				ctx.RenderPage(model, "templates/layout.html", "templates/share_file_detail.html")
			}

		}
	})
	root.RegexMatch(regexp.MustCompile(Path_Download_File+`/.+`), s.downloadHandler())
	root.GET(Path_Index, s.indexHandler())
	root.GET(Path_File, s.fileDetailsHandler())
	root.GET(Path_File_List, s.fileListHandler())
	open.GET(Path_Login, s.loginHandler())
	open.GET(Path_Login_Callback, s.loginCallbackHandler())
	root.POST(Path_Logout, s.logoutHandler())
	root.DELETE(Path_File, s.fileDeleteHandler())
	root.GET(Path_Server, s.serverHandler())
	root.DELETE(Path_Server, s.serverDeleteHandler())
	root.GET(Path_Server_Edit, s.serverEditHandler())
	root.POST(Path_Server_Edit, s.serverEditPostHandler())
	root.DELETE(Path_Directory, s.directoryDeleteHandler())
	root.GET(Path_File_Edit, s.fileEditHandler())
	root.POST(Path_File_Edit, s.fileEditPostHandler())
	root.GET(Path_File_Move, s.fileMoveHandler())
	root.POST(Path_File_Move, s.fileMovePostHandler())
	root.GET(Path_File_Copy, s.fileCopyHandler())
	root.POST(Path_File_Copy, s.fileCopyPostHandler())
	root.GET(Path_File_History, s.fileHistoryHandler())
	root.GET(Path_File_Rename, s.fileRenameHandler())
	root.POST(Path_File_Rename, s.fileRenamePostHandler())
	open.POST(Path_File_Upload, s.fileUploadPostHandler())
	root.POST(Path_File_Share, s.fileSharePostHandler())
	root.GET(Path_File_Commit, s.fileCommitHandler())
	root.GET(Path_File_Commit_Detail, s.fileCommitDetailHandler())
	open.GET(Path_QRCode, s.qrCodeHandler())
}

//dl value: 0 not download, 1 download
func (s *FileSyncWebServer) generateShareUrl(path string, token string, dl string) string {
	if dl != "1" && dl != "0" {
		panic("parameter error:dl")
	}
	return "/sh/" + token + path + "?dl=" + dl
}
func (s *FileSyncWebServer) fileSharePostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		path := ctx.Request.FormValue("path")
		commit_id := ctx.Request.FormValue("commit_id")
		max_commit_id := ctx.Request.FormValue("max_commit_id")
		partition_id := s.MustGetLoginUser(ctx).Partition_id
		if max_commit_id == "" {
			commit := s.GetStorage(ctx).GetPartitionLatestCommit(partition_id)
			max_commit_id = commit["id"].(string)
		}
		token := s.GetStorage(ctx).AddShare(path, partition_id, commit_id, max_commit_id, s.MustGetLoginUser(ctx).Id)
		ctx.Success(s.generateShareUrl("", token, "0"))
	}
}

func (s *FileSyncWebServer) fileRenameHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.FormValue("id")
		file := s.GetStorage(ctx).GetFile(id)
		model := struct {
			File models.File
		}{File: file}
		ctx.RenderPage(s.newPageModel(ctx, model), "templates/layout.html", "templates/file_rename.html")
	}
}

func (s *FileSyncWebServer) fileRenamePostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.FormValue("id")
		name := ctx.Request.FormValue("name")
		actions := []storage.Action{}
		action := storage.RenameAction{Id: id, NewName: name}
		actions = append(actions, action)
		if err := s.GetStorage(ctx).SuperDoFileActions(actions, s.MustGetLoginUser(ctx).Id, s.MustGetLoginUser(ctx).Partition_id); err != nil {
			panic(err)
		}
		ctx.Success(nil)
	}
}
func (s *FileSyncWebServer) fileHistoryHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		path := ctx.Request.FormValue("path")
		histories := s.GetStorage(ctx).GetHistoryRevisions(path, s.MustGetLoginUser(ctx).Partition_id, common.MaxInt64)
		fmt.Println(histories)
		model := struct {
			Path      string
			Histories []map[string]interface{}
		}{Histories: histories, Path: path}
		ctx.FuncMap["detailUrl"] = func(file map[string]interface{}) (string, error) {
			if file["type"] == "1" {
				parameters := url.Values{}
				parameters.Add("path", filepath.Join("/", path))
				parameters.Add("commit_id", file["commit_id"].(string))
				return Path_File + "?" + parameters.Encode(), nil
			} else {
				parameters := url.Values{}
				parameters.Add("path", filepath.Join("/", path))
				parameters.Add("commit_id", file["commit_id"].(string))
				return Path_File_List + "?" + parameters.Encode(), nil
			}
		}
		ctx.RenderPage(s.newPageModel(ctx, model), "templates/layout.html", "templates/file_history.html")
	}
}
func (s *FileSyncWebServer) fileEditHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.RenderPage(s.newPageModel(ctx, nil), "templates/layout.html", "templates/file_edit.html")
	}
}
func (s *FileSyncWebServer) fileEditPostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		path := ctx.Request.FormValue("path")
		actions := []storage.Action{}
		action := storage.CreateDirectoryAction{Path: path}
		actions = append(actions, action)
		if err := s.GetStorage(ctx).SuperDoFileActions(actions, s.MustGetLoginUser(ctx).Id, s.MustGetLoginUser(ctx).Partition_id); err != nil {
			panic(err)
		}
		ctx.Success(nil)
	}
}
func (s *FileSyncWebServer) fileMoveHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.RenderPage(s.newPageModel(ctx, nil), "templates/layout.html", "templates/file_move.html")
	}
}
func (s *FileSyncWebServer) fileMovePostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.FormValue("id")
		destination := ctx.Request.FormValue("destination")
		actions := []storage.Action{}
		action := storage.MoveAction{Id: id, DestinationPath: destination}
		actions = append(actions, action)
		if err := s.GetStorage(ctx).SuperDoFileActions(actions, s.MustGetLoginUser(ctx).Id, s.MustGetLoginUser(ctx).Partition_id); err != nil {
			panic(err)
		}
		ctx.Success(nil)
	}
}

func (s *FileSyncWebServer) fileCopyHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.RenderPage(s.newPageModel(ctx, nil), "templates/layout.html", "templates/file_copy.html")
	}
}
func (s *FileSyncWebServer) fileCopyPostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.FormValue("id")
		destination := ctx.Request.FormValue("destination")
		actions := []storage.Action{}
		action := storage.CopyAction{Id: id, DestinationPath: destination}
		actions = append(actions, action)
		if err := s.GetStorage(ctx).SuperDoFileActions(actions, s.MustGetLoginUser(ctx).Id, s.MustGetLoginUser(ctx).Partition_id); err != nil {
			panic(err)
		}
		ctx.Success(nil)
	}
}
func (s *FileSyncWebServer) serverHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		servers := s.GetStorage(ctx).GetServers()
		ctx.FuncMap["editUrl"] = func(id string) (string, error) {
			p := ""
			if id != "" {
				p = "?id=" + id
			}
			return Path_Server_Edit + p, nil
		}
		ctx.RenderPage(s.newPageModel(ctx, servers), "templates/layout.html", "templates/server.html")
	}
}
func (s *FileSyncWebServer) directoryDeleteHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.FormValue("id")
		actions := []storage.Action{}
		action := storage.DeleteAction{Id: id}
		actions = append(actions, action)
		if err := s.GetStorage(ctx).SuperDoFileActions(actions, s.MustGetLoginUser(ctx).Id, s.MustGetLoginUser(ctx).Partition_id); err != nil {
			panic(err)
		}
		ctx.Success(nil)
	}
}
func (s *FileSyncWebServer) serverDeleteHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.FormValue("id")
		s.GetStorage(ctx).DeleteServer(id)
		ctx.Success(nil)
	}
}
func (s *FileSyncWebServer) serverEditPostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.FormValue("id")
		name := ctx.Request.FormValue("name")
		ip := ctx.Request.FormValue("ip")
		port := ctx.Request.FormValue("port")
		if id == "" {
			//add server
			s.GetStorage(ctx).AddServer(name, ip, port)
		} else {
			//update server
			s.GetStorage(ctx).UpdateServer(id, name, ip, port)
		}
		ctx.Success(nil)
	}
}
func (s *FileSyncWebServer) serverEditHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		server_id := ctx.Request.FormValue("id")
		server := &models.Server{}
		if server_id != "" {
			server = s.GetStorage(ctx).GetServer(server_id)
			if server == nil {
				panic("server not found")
			}
		}
		ctx.RenderPage(s.newPageModel(ctx, map[string]interface{}{"DeleteUrl": Path_Server + "?id=" + server.Id, "server": server}), "templates/layout.html", "templates/server_edit.html")
	}
}
func (s *FileSyncWebServer) genericMiddleware() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		if strings.Index(ctx.Request.URL.Path, Path_Download_File) != 0 {
			ctx.Writer.EnsureInitialzed(true)
		}
		if session, err := auth.GetSessionByToken(s.rac, ctx, s.oAuth2Config, s.config.OAuth.IntrospectTokenURL, s.skip_tls_verify); err == nil {
			user := s.GetStorage(ctx).GetUserByOpId(session.Claims["sub"].(string))
			ctx.Data["user"] = user
		} else {
			http.Redirect(ctx.Writer, ctx.Request, Path_Login, 302)
		}
	}
}
func (s *FileSyncWebServer) authenticateHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
	}
}
func (s *FileSyncWebServer) indexHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		http.Redirect(ctx.Writer, ctx.Request, Path_File_List+"?path=/", 302)
	}
}

func (s *FileSyncWebServer) fileDeleteHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		id := ctx.Request.FormValue("id")
		actions := []storage.Action{}
		action := storage.DeleteAction{Id: id}
		actions = append(actions, action)
		if err := s.GetStorage(ctx).SuperDoFileActions(actions, s.MustGetLoginUser(ctx).Id, s.MustGetLoginUser(ctx).Partition_id); err != nil {
			panic(err)
		}
		ctx.Success(nil)
	}
}
func (s *FileSyncWebServer) fileListHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		path := ctx.Request.FormValue("path")
		commit_id := ctx.Request.FormValue("commit_id")
		max_commit_id := ctx.Request.FormValue("max")
		files, err := s.GetStorage(ctx).GetFiles(path, commit_id, max_commit_id, s.MustGetLoginUser(ctx).Partition_id)
		if err != nil {
			panic(err)
		}
		data := struct {
			Path             string
			Files            []map[string]interface{}
			DirectoryUrlPath string
			ShareUrlPath     string
			Path_File_Edit   string
			Path_File_Move   string
			Path_File_Copy   string
			Path_File_Rename string
			File_Path        string
		}{Path: path, Files: files, DirectoryUrlPath: Path_Directory, Path_File_Edit: Path_File_Edit, Path_File_Move: Path_File_Move, Path_File_Copy: Path_File_Copy, File_Path: filepath.Join("/", path), Path_File_Rename: Path_File_Rename, ShareUrlPath: Path_File_Share}
		ctx.FuncMap["detailUrl"] = func(file map[string]interface{}) (string, error) {
			if file["type"] == "1" {
				parameters := url.Values{}
				parameters.Add("path", filepath.Join("/", path, file["name"].(string)))
				parameters.Add("commit_id", file["commit_id"].(string))
				return Path_File + "?" + parameters.Encode(), nil
			} else {
				parameters := url.Values{}
				parameters.Add("path", filepath.Join("/", path, file["name"].(string)))
				parameters.Add("commit_id", file["commit_id"].(string))
				parameters.Add("max", max_commit_id)
				return Path_File_List + "?" + parameters.Encode(), nil
			}
		}
		ctx.FuncMap["isHidden"] = func(isHidden bool) (string, error) {
			if isHidden {
				return "true", nil
			} else {
				return "false", nil
			}
		}
		model := s.newPageModel(ctx, data)
		model.PageTitle = filepath.Join("/", path)
		ctx.RenderPage(model, "templates/layout.html", "templates/file_list.html")
	}
}
func (s *FileSyncWebServer) fileDetailsHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		login_user, _ := s.GetLoginUser(ctx)
		if login_user == nil {
			panic("no logged user")
		}
		path := ctx.Request.FormValue("path")
		commit_id := ctx.Request.FormValue("commit_id")
		histories := s.GetStorage(ctx).GetHistoryRevisions(path, login_user.Partition_id, common.MaxInt64)
		var m_file map[string]interface{} = nil
		var m_file_latest map[string]interface{} = nil
		var latest_revision_url = ""
		if len(histories) > 0 {
			m_file = histories[0]
			m_file_latest = histories[0]
		} else if m_file == nil {
			panic("the file does not exist")
		}
		if commit_id != "" {
			for _, his := range histories {
				if his["commit_id"].(string) == commit_id {
					m_file = his
					break
				}
			}
		}
		if m_file_latest["commit_id"].(string) != m_file["commit_id"].(string) {
			parameters := url.Values{}
			parameters.Add("path", filepath.Join("/", path))
			parameters.Add("commit_id", m_file_latest["commit_id"].(string))
			latest_revision_url = Path_File + "?" + parameters.Encode()
		}
		id := m_file["id"].(string)
		server_file := s.GetStorage(ctx).GetServerFileByFileId(id)
		file := s.GetStorage(ctx).GetFile(id)
		if server_file == nil {
			panic("file not found")
		}
		ctx.FuncMap["downloadUrl"] = func() (string, error) {
			return Path_Download_File + "/" + id + "/" + server_file.Name, nil
		}
		can_delete := false
		if login_user != nil && login_user.Id == file.User_id {
			can_delete = true
		}
		p := url.Values{}
		p.Add("path", path)
		model := struct {
			DownloadUrl         string
			DeleteUrl           string
			FileId              string
			File                models.File
			ServerFile          models.ServerFile
			History             map[string]interface{}
			CanDelete           bool
			HistoryUrl          string
			Latest_revision_url string
		}{DownloadUrl: Path_Download_File + "/" + id + "/" + server_file.Name, DeleteUrl: Path_File + "?id=" + id, File: file, ServerFile: *server_file, FileId: id, CanDelete: can_delete, HistoryUrl: Path_File_History + "?" + p.Encode(), Latest_revision_url: latest_revision_url, History: m_file}
		ctx.RenderPage(s.newPageModel(ctx, model), "templates/layout.html", "templates/file_details.html")
	}
}

func (s *FileSyncWebServer) loginHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		if ctx.Request.URL.Query().Get("native") == "1" {
			s.oAuth2Config.RedirectURL = s.config.OAuth.NativeAppRedirectURL
		} else {
			s.oAuth2Config.RedirectURL = s.config.OAuth.RedirectURL
		}
		url := s.oAuth2Config.AuthCodeURL("state-string", oauth2.AccessTypeOffline)
		http.Redirect(ctx.Writer, ctx.Request, url, 302)
	}
}
func (s *FileSyncWebServer) addOrUpdateUser(ctx *goweb.Context, token *oauth2.Token) {
	rar := common.NewRestApiRequest("GET", s.config.OAuth.UserInfoURL, nil).SetAuthHeader(token)
	resp, err := s.rac.Do(rar)
	if err != nil {
		panic(err)
	}
	m, err := common.ReadAsMap(resp.Body)
	if err != nil {
		panic(err)
	}
	data := m["data"].(map[string]interface{})
	sub := data["Id"].(string)
	name := data["Name"].(string)
	_, err = s.GetStorage(ctx).AddOrUpdateUser(sub, name)
	if err != nil {
		panic(err)
	}
}
func (s *FileSyncWebServer) loginCallbackHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		code := ctx.Request.URL.Query().Get("code")
		token, err := s.oAuth2Config.Exchange(context.WithValue(context.Background(), "", s.httpClient), code)
		if err != nil {
			panic(err)
		}
		auth.Login(ctx, token, s.config.OAuth.JWKJsonUrl)
		http.Redirect(ctx.Writer, ctx.Request, Path_Index, 302)
		s.addOrUpdateUser(ctx, token)
	}
}

func (s *FileSyncWebServer) logoutHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		auth.Logout(s.rac, ctx, s.oAuth2Config, s.config.OAuth.IntrospectTokenURL, s.skip_tls_verify, func(id_token string) {
			parameters := url.Values{}
			parameters.Add("id_token_hint", id_token)
			redirect_url := s.config.OAuth.LogoutRedirectUrl
			parameters.Add("post_logout_redirect_uri", redirect_url)
			http.Redirect(ctx.Writer, ctx.Request, s.config.OAuth.LogoutUrl+"?"+parameters.Encode(), 302)
		})
	}
}

func (s *FileSyncWebServer) fileUploadPostHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		err := ctx.Request.ParseMultipartForm(1024 * 10)
		if err != nil {
			panic(err)
		}
		md5 := ctx.Request.Form.Get("md5")
		if md5 == "" {
			panic("missing md5 parameter")
		}
		file, header, err := ctx.Request.FormFile("file")
		if err != nil {
			panic(err)
		}
		temp_path := filepath.Join(s.config.temp_folder, md5)
		temp_file, err := os.Create(temp_path)
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(temp_file, file)
		if err != nil {
			panic(err)
		}
		err = temp_file.Close()
		if err != nil {
			panic(err)
		}
		temp_md5, err := common.FileMd5Hash(temp_path)
		if err != nil {
			panic(err)
		}
		if strings.ToUpper(temp_md5) != md5 {
			panic("the md5 of uploaded file and the md5 paramter value are inconsistent")
		}
		fileName, err := url.QueryUnescape(header.Filename)
		if err != nil {
			panic(err)
		}
		filepath := filepath.Join(s.config.upload_folder, md5+"-"+fileName)

		err = os.Rename(temp_path, filepath)
		if err != nil {
			panic(err)
		}
		log.Println("received file:", fileName)
	}
}
func (s *FileSyncWebServer) downloadHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		fmt.Println(ctx.Request.Header)
		segments := strings.Split(ctx.Request.URL.Path, "/")
		file_id := segments[3]
		file_name := segments[4]
		server_file := s.GetStorage(ctx).GetServerFileByFileId(file_id)
		if file_name != server_file.Name {
			panic("file not found")
		}
		conn, err := net.Dial("tcp", server_file.Ip+":"+strconv.Itoa(server_file.Port))
		if err != nil {
			panic(err)
		}
		s := session.NewSession(conn)
		msg := message.NewMessage(message.MT_Download_File)
		msg.Header["path"] = server_file.Path
		_, err = s.Fetch(msg, nil)
		if err != nil {
			panic(err)
		}
		ctx.Writer.Header().Set("Content-Type", "application/octet-stream")
		ctx.Writer.Header().Set("Content-Disposition", `attachment; filename="`+server_file.Name+`"`)
		ctx.Writer.Header().Set("Content-Length", strconv.FormatInt(server_file.Size, 10))
		if ctx.Writer.Compress {
			panic("compression should not pick up")
		}
		_, err = io.CopyN(ctx.Writer, s, server_file.Size)
		if err != nil {
			log.Println(err)
			panic(err)
		}
		s.Close()
	}
}

func (s *FileSyncWebServer) fileCommitHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		commits := s.GetStorage(ctx).GetRecentCommits(s.MustGetLoginUser(ctx).Partition_id)
		model := struct {
			Commits []map[string]interface{}
		}{Commits: commits}
		ctx.FuncMap["detailUrl"] = func(file map[string]interface{}) (string, error) {
			return Path_File_Commit_Detail + "?id=" + file["id"].(string), nil
		}
		ctx.RenderPage(s.newPageModel(ctx, model), "templates/layout.html", "templates/file_commit.html")
	}
}

type FileChange struct {
	Id          string
	Path        string
	ChangeType  int //1 add,2 delete,3 move,4 rename,5 copy
	Source_Path string
}

func (s *FileSyncWebServer) fileCommitDetailHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		commit_id := ctx.Request.FormValue("id")
		user := s.MustGetLoginUser(ctx)
		commit := s.GetStorage(ctx).GetCommitById(commit_id)
		commit_index, err := strconv.ParseInt(commit["index"].(string), 10, 64)
		if err != nil {
			panic(err)
		}
		changes := s.GetStorage(ctx).GetCommitChanges(user.Partition_id, commit_id)
		file_changes := []FileChange{}
		for _, v := range changes {
			change := FileChange{}
			id := v["id"].(string)
			change.Id = id
			var path string
			if commit_id == v["start_commit_id"].(string) {
				path = s.GetStorage(ctx).GetFilePath(user.Partition_id, id, commit_index)
				change.ChangeType = 1
			} else if commit_id == v["end_commit_id"].(string) {
				path = s.GetStorage(ctx).GetFilePath(user.Partition_id, id, commit_index-1)
				change.ChangeType = 2
			}
			change.Path = path
			if change.ChangeType == 1 && v["source"] != nil {
				for i, v2 := range file_changes {
					if v2.Id == v["source"].(string) {
						file_changes = append(file_changes[:i], file_changes[i+1:]...)
						change.Source_Path = v2.Path
						if filepath.Dir(change.Path) == filepath.Dir(v2.Path) {
							change.ChangeType = 4
						} else {
							change.ChangeType = 3
						}
						break
					}
				}
				if change.Source_Path == "" {
					change.Source_Path = s.GetStorage(ctx).GetFilePath(user.Partition_id, v["source"].(string), commit_index-1)
					change.ChangeType = 5
				}
			}
			file_changes = append(file_changes, change)
		}
		model := struct {
			Str     string
			Changes []FileChange
		}{Str: fmt.Sprintln(changes), Changes: file_changes}
		ctx.RenderPage(s.newPageModel(ctx, model), "templates/layout.html", "templates/file_commit_detail.html")
	}
}

func (s *FileSyncWebServer) qrCodeHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		str := ctx.Request.FormValue("str")
		qrCode, _ := qr.Encode(str, qr.L, qr.Auto)
		qrCode, _ = barcode.Scale(qrCode, 300, 300)
		png.Encode(ctx.Writer, qrCode)
	}
}

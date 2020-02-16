package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

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
	Path_Index          = "/"
	Path_File           = "/file"
	Path_File_List      = "/file/list"
	Path_Login          = "/login"
	Path_Login_Callback = "/login-callback"
	Path_Logout         = "/logout"
	Path_Download_File  = "/file/download"
	Path_Server         = "/server"
	Path_Server_Edit    = "/server_edit"
	Path_Directory      = "/directory"
)

func (s *FileSyncWebServer) bindHandlers(root *goweb.RouterGroup) {
	root.RegexMatch(regexp.MustCompile(Path_Download_File+`/.+`), s.downloadHandler())
	root.Use(s.genericMiddleware())
	root.RegexMatch(regexp.MustCompile(`/static/.+`), func(context *goweb.Context) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))).ServeHTTP(context.Writer, context.Request)
	})
	root.GET(Path_Index, s.indexHandler())
	root.GET(Path_File, s.fileDetailsHandler())
	root.GET(Path_File_List, s.fileListHandler())
	root.GET(Path_Login, s.loginHandler())
	root.GET(Path_Login_Callback, s.loginCallbackHandler())
	root.POST(Path_Logout, s.logoutHandler())
	root.DELETE(Path_File, s.fileDeleteHandler())
	root.GET(Path_Server, s.serverHandler())
	root.DELETE(Path_Server, s.serverDeleteHandler())
	root.GET(Path_Server_Edit, s.serverEditHandler())
	root.POST(Path_Server_Edit, s.serverEditPostHandler())
	root.DELETE(Path_Directory, s.directoryDeleteHandler())
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
		s.GetStorage(ctx).DeleteDirectory(id)
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
		ctx.Writer.EnsureInitialzed(true)
		if session, err := auth.GetSessionByToken(s.rac, ctx, s.oAuth2Config, s.config.OAuth.IntrospectTokenURL, s.skip_tls_verify); err == nil {
			user := s.GetStorage(ctx).GetUserByOpId(session.Claims["sub"].(string))
			ctx.Data["user"] = user
		}
	}
}
func (s *FileSyncWebServer) authenticateHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
	}
}
func (s *FileSyncWebServer) indexHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		directory := s.GetStorage(ctx).GetDirectory("", s.MustGetLoginUser(ctx).Id)
		files := s.GetStorage(ctx).GetFiles(directory.Id)
		data := struct {
			Files []models.File
		}{Files: files}
		ctx.FuncMap["detailUrl"] = func(id string) (string, error) {
			return Path_File + "?id=" + id, nil
		}
		ctx.RenderPage(s.newPageModel(ctx, data), "templates/layout.html", "templates/index.html")
	}
}

func (s *FileSyncWebServer) fileDeleteHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		file_id := ctx.Request.FormValue("file_id")
		s.GetStorage(ctx).DeleteFile(file_id)
		ctx.Success(nil)
	}
}
func (s *FileSyncWebServer) fileListHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		path := ctx.Request.FormValue("path")
		directory := s.GetStorage(ctx).GetDirectory(path, s.MustGetLoginUser(ctx).Id)
		files := s.GetStorage(ctx).GetFiles(directory.Id)
		directories := s.GetStorage(ctx).GetDirectories(directory.Id)
		data := struct {
			Path             string
			Files            []models.File
			Directories      []models.Directory
			DirectoryUrlPath string
		}{Path: path, Files: files, Directories: directories, DirectoryUrlPath: Path_Directory}
		ctx.FuncMap["detailUrl"] = func(id string) (string, error) {
			return Path_File + "?id=" + id, nil
		}
		ctx.FuncMap["directoryUrl"] = func(dir_name string) (string, error) {
			p := path + "/" + dir_name
			p = strings.TrimPrefix(p, "/")
			parameters := url.Values{}
			parameters.Add("path", p)
			return Path_File_List + "?" + parameters.Encode(), nil
		}
		ctx.FuncMap["isHidden"] = func(isHidden bool) (string, error) {
			if isHidden {
				return "true", nil
			} else {
				return "false", nil
			}
		}
		model := s.newPageModel(ctx, data)
		model.PageTitle = "/" + path
		ctx.RenderPage(model, "templates/layout.html", "templates/file_list.html")
	}
}
func (s *FileSyncWebServer) fileDetailsHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		file_id := ctx.Request.FormValue("id")
		server_file := s.GetStorage(ctx).GetServerFileByFileId(file_id)
		if server_file == nil {
			panic("file not found")
		}
		ctx.FuncMap["downloadUrl"] = func() (string, error) {
			return Path_Download_File + "/" + file_id + "/" + server_file.Name, nil
		}
		model := struct {
			DownloadUrl string
			DeleteUrl   string
			FileId      string
			ServerFile  models.ServerFile
		}{DownloadUrl: Path_Download_File + "/" + file_id + "/" + server_file.Name, DeleteUrl: Path_File + "?file_id=" + file_id, ServerFile: *server_file, FileId: file_id}
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
	rar := common.NewRestApiRequest("GET", s.config.OAuth.UserInfoURL, nil).UseToken(s.oAuth2Config, token)
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
	s.GetStorage(ctx).AddOrUpdateUser(sub, name)
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

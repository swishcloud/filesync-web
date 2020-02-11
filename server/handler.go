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

func (s *FileSyncWebServer) GetLoginUser(ctx *goweb.Context) (*models.User, error) {
	var sub string
	if tokenstr, err := auth.GetBearerToken(ctx); err != nil {
		if s, err := auth.GetSessionByToken(ctx, s.oAuth2Config, s.config.OAuth.IntrospectTokenURL, s.skip_tls_verify); err != nil {
			return nil, err
		} else {
			sub = s.Claims["sub"].(string)
		}
	} else {
		token := &oauth2.Token{AccessToken: tokenstr}
		if sub, err = auth.CheckToken(s.oAuth2Config, token, s.config.OAuth.IntrospectTokenURL, s.skip_tls_verify); err != nil {
			return nil, err
		}
	}
	user := s.GetStorage(ctx).GetUserByOpId(sub)
	return user, nil
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
	u, _ := s.GetLoginUser(ctx)
	m.User = u
	m.WebsiteName = "filesync-web"
	return m
}

const (
	Path_Index          = "/"
	Path_File           = "/file"
	Path_Login          = "/login"
	Path_Login_Callback = "/login-callback"
	Path_Logout         = "/logout"
	Path_Download_File  = "/file/download"
)

func (s *FileSyncWebServer) bindHandlers(root goweb.RouterGroup) {
	root.RegexMatch(regexp.MustCompile(Path_Download_File+`/.+`), s.downloadHandler())
	root.Use(s.genericMiddleware())
	root.RegexMatch(regexp.MustCompile(`/static/.+`), func(context *goweb.Context) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))).ServeHTTP(context.Writer, context.Request)
	})
	root.GET(Path_Index, s.indexHandler())
	root.GET(Path_File, s.fileDetailsHandler())
	root.GET(Path_Login, s.loginHandler())
	root.GET(Path_Login_Callback, s.loginCallbackHandler())
	root.POST(Path_Logout, s.logoutHandler())
	root.DELETE(Path_File, s.fileDeleteHandler())
}
func (s *FileSyncWebServer) genericMiddleware() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.Writer.EnsureInitialzed(true)
	}
}
func (s *FileSyncWebServer) authenticateHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
	}
}
func (s *FileSyncWebServer) indexHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		files := s.GetStorage(ctx).GetAllFiles()
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

func (s *FileSyncWebServer) loginCallbackHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		code := ctx.Request.URL.Query().Get("code")
		token, err := s.oAuth2Config.Exchange(context.WithValue(context.Background(), "", s.httpClient), code)
		if err != nil {
			panic(err)
		}
		session := auth.Login(ctx, token, s.config.OAuth.JWKJsonUrl)
		http.Redirect(ctx.Writer, ctx.Request, Path_Index, 302)

		rac := common.NewRestApiClient("GET", s.config.OAuth.UserInfoURL, nil, s.skip_tls_verify).UseToken(s.oAuth2Config, session.GetToken())
		resp, err := rac.Do()
		if err != nil {
			panic(err)
		}
		m := common.ReadAsMap(resp.Body)
		data := m["data"].(map[string]interface{})
		sub := data["Id"].(string)
		name := data["Name"].(string)
		s.GetStorage(ctx).AddOrUpdateUser(sub, name)
		log.Println(m)
	}
}

func (s *FileSyncWebServer) logoutHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		auth.Logout(ctx, s.oAuth2Config, s.config.OAuth.IntrospectTokenURL, s.skip_tls_verify, func(id_token string) {
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

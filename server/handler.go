package server

import (
	"context"
	"net/http"
	"regexp"

	"github.com/swishcloud/filesync-web/storage/models"
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

func GetLoginUser(ctx *goweb.Context) (*models.User, error) {
	if s, err := auth.GetSessionByToken(ctx); err != nil {
		return nil, err
	} else {
		u := &models.User{}
		u.Id = s.Claims["sub"].(string)
		u.Name = s.Claims["name"].(string)
		avartar := s.Claims["avatar"].(string)
		u.Avatar = &avartar
		return u, nil
	}
}
func newPageModel(ctx *goweb.Context, data interface{}) pageModel {
	m := pageModel{}
	m.Data = data
	m.MobileCompatible = true
	u, _ := GetLoginUser(ctx)
	m.User = u
	m.WebsiteName = "filesync-web"
	return m
}

const (
	Path_Index          = "/"
	Path_File_Details   = "/file-details"
	Path_Login          = "/login"
	Path_Login_Callback = "/login-callback"
)

func (s *FileSyncWebServer) bindHandlers(root goweb.RouterGroup) {
	root.RegexMatch(regexp.MustCompile(`/static/.+`), func(context *goweb.Context) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))).ServeHTTP(context.Writer, context.Request)
	})
	root.GET(Path_Index, s.indexHandler())
	root.GET(Path_File_Details, s.fileDetailsHandler())
	root.GET(Path_Login, s.loginHandler())
	root.GET(Path_Login_Callback, s.loginCallbackHandler())
}
func (s *FileSyncWebServer) indexHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		files := s.storage.GetAllFiles()
		ctx.RenderPage(newPageModel(ctx, files), "templates/layout.html", "templates/index.html")
	}
}

func (s *FileSyncWebServer) fileDetailsHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.RenderPage(newPageModel(ctx, nil), "templates/layout.html", "templates/file_details.html")
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
			ctx.ShowErrorPage(http.StatusBadRequest, err.Error())
			return
		}
		auth.Login(ctx, token, s.config.OAuth.JWKJsonUrl)
		http.Redirect(ctx.Writer, ctx.Request, Path_Index, 302)
	}
}

package server

import (
	"net/http"
	"regexp"

	"github.com/swishcloud/filesync-web/storage/models"
	"github.com/swishcloud/goweb"
	"github.com/swishcloud/goweb/auth"
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
	Path_Index        = "/"
	Path_File_Details = "/file-details"
)

func (s *FileSyncWebServer) bindHandlers(root goweb.RouterGroup) {
	root.RegexMatch(regexp.MustCompile(`/static/.+`), func(context *goweb.Context) {
		http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))).ServeHTTP(context.Writer, context.Request)
	})
	root.GET(Path_Index, s.indexHandler())
	root.GET(Path_File_Details, s.fileDetailsHandler())
}
func (s *FileSyncWebServer) indexHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.RenderPage(newPageModel(ctx, nil), "templates/layout.html", "templates/index.html")
	}
}

func (s *FileSyncWebServer) fileDetailsHandler() goweb.HandlerFunc {
	return func(ctx *goweb.Context) {
		ctx.RenderPage(newPageModel(ctx, nil), "templates/layout.html", "templates/file_details.html")
	}
}

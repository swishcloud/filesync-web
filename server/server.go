package server

import (
	"crypto/tls"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"

	"gopkg.in/yaml.v2"

	"github.com/swishcloud/filesync-web/storage"
	"github.com/swishcloud/goweb"
)

type Config struct {
	FILE_LOCATION string      `yaml:"file_location"`
	DB_CONN_INFO  string      `yaml:"db_conn_info"`
	OAuth         ConfigOAuth `yaml:"oauth"`
	upload_folder string
	Tls_cert_file string `yaml:"tls_cert_file"`
	Tls_key_file  string `yaml:"tls_key_file"`
}
type ConfigOAuth struct {
	ClientId             string `yaml:"ClientId"`
	TokenUrl             string `yaml:"TokenUrl"`
	AuthUrl              string `yaml:"AuthUrl"`
	Secret               string `yaml:"Secret"`
	RedirectURL          string `yaml:"RedirectURL"`
	NativeAppRedirectURL string `yaml:"NativeAppRedirectURL"`
	IntrospectTokenURL   string `yaml:"IntrospectTokenURL"`
	LogoutUrl            string `yaml:"LogoutUrl"`
	LogoutRedirectUrl    string `yaml:"LogoutRedirectUrl"`
	JWKJsonUrl           string `yaml:"JWKJsonUrl"`
	UserInfoURL          string `yaml:"UserInfoURL"`
}
type FileSyncWebServer struct {
	engine          *goweb.Engine
	config          *Config
	oAuth2Config    *oauth2.Config
	skip_tls_verify bool
	httpClient      *http.Client
}

func NewFileSyncWebServer(configPath string, skip_tls_verify bool) *FileSyncWebServer {
	s := new(FileSyncWebServer)
	s.skip_tls_verify = skip_tls_verify
	s.httpClient = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: skip_tls_verify}}}
	http.DefaultClient = s.httpClient
	s.engine = goweb.Default()
	s.engine.WM.HandlerWidget = &HandlerWidget{s: s}
	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}
	s.config = new(Config)
	err = yaml.Unmarshal(b, s.config)
	if err != nil {
		log.Fatal(err)
	}

	s.config.upload_folder = s.config.FILE_LOCATION + "upload/"
	err = os.MkdirAll(s.config.upload_folder, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	s.oAuth2Config = &oauth2.Config{
		ClientID:     s.config.OAuth.ClientId,
		ClientSecret: s.config.OAuth.Secret,
		Scopes:       []string{"offline", "openid", "profile"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  s.config.OAuth.AuthUrl,
			TokenURL: s.config.OAuth.TokenUrl,
		},
	}
	return s
}
func (s *FileSyncWebServer) Serve() {
	s.bindHandlers(s.engine.RouterGroup)
	apiGroup := s.engine.RouterGroup.Group()
	s.bindApiHandlers(apiGroup)
	addr := ":2002"
	log.Println("listening on", addr)
	err := http.ListenAndServeTLS(addr, s.config.Tls_cert_file, s.config.Tls_key_file, s.engine)
	if err != nil {
		log.Fatal(err)
	}
}
func (server *FileSyncWebServer) GetStorage(ctx *goweb.Context) storage.Storage {
	m := ctx.Data["storage"]
	if m == nil {
		m = storage.NewSQLManager(server.config.DB_CONN_INFO)
		ctx.Data["storage"] = m
	}
	return m.(storage.Storage)
}

type HandlerWidget struct {
	s *FileSyncWebServer
}

func (*HandlerWidget) Pre_Process(ctx *goweb.Context) {
}
func (hw *HandlerWidget) Post_Process(ctx *goweb.Context) {
	m := ctx.Data["storage"]
	if m != nil {
		if ctx.Ok {
			m.(storage.Storage).Commit()
		} else {
			m.(storage.Storage).Rollback()
		}
	}

	if ctx.Err != nil {
		data := struct {
			Desc string
		}{Desc: ctx.Err.Error()}
		model := hw.s.newPageModel(ctx, data)
		model.PageTitle = "ERROR"
		ctx.RenderPage(model, "templates/layout.html", "templates/error.html")
	}

}

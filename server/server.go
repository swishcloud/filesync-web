package server

import (
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/oauth2"

	"gopkg.in/yaml.v2"

	"github.com/swishcloud/filesync-web/storage"
	"github.com/swishcloud/goweb"
)

type Config struct {
	DB_CONN_INFO string      `yaml:"db_conn_info"`
	OAuth        ConfigOAuth `yaml:"oauth"`
}
type ConfigOAuth struct {
	ClientId          string `yaml:"ClientId"`
	TokenUrl          string `yaml:"TokenUrl"`
	AuthUrl           string `yaml:"AuthUrl"`
	Secret            string `yaml:"Secret"`
	LogoutUrl         string `yaml:"LogoutUrl"`
	LogoutRedirectUrl string `yaml:"LogoutRedirectUrl"`
	JWKJsonUrl        string `yaml:"JWKJsonUrl"`
}
type FileSyncWebServer struct {
	engine       *goweb.Engine
	storage      storage.Storage
	config       *Config
	oAuth2Config *oauth2.Config
}

func NewFileSyncWebServer() *FileSyncWebServer {
	s := new(FileSyncWebServer)
	s.engine = goweb.Default()
	b, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	s.config = new(Config)
	err = yaml.Unmarshal(b, s.config)
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

	s.storage = storage.NewSQLManager(s.config.DB_CONN_INFO)
	return s
}
func (s *FileSyncWebServer) Serve() {
	s.bindHandlers(s.engine.RouterGroup)
	addr := ":2002"
	log.Println("listening on", addr)
	err := http.ListenAndServe(addr, s.engine)
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

package server

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/swishcloud/filesync/message"

	"golang.org/x/oauth2"

	"gopkg.in/yaml.v2"

	"github.com/swishcloud/filesync-web/storage"
	"github.com/swishcloud/filesync-web/storage/models"
	"github.com/swishcloud/filesync/session"
	"github.com/swishcloud/gostudy/common"
	"github.com/swishcloud/goweb"
)

type Config struct {
	Listen_ip      string      `yaml:"listen_ip"`
	Website_domain string      `yaml:"website_domain"`
	FILE_LOCATION  string      `yaml:"file_location"`
	DB_CONN_INFO   string      `yaml:"db_conn_info"`
	OAuth          ConfigOAuth `yaml:"oauth"`
	upload_folder  string
	temp_folder    string
	Tls_cert_file  string `yaml:"tls_cert_file"`
	Tls_key_file   string `yaml:"tls_key_file"`
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
	rac             *common.RestApiClient
}
type TcpServer struct {
	listenPort int
	clients    []*client
	connect    chan *session.Session
	disconnect chan *session.Session
	config     *Config
}
type client struct {
	session *session.Session
	class   int
}

func readConfig(configPath string) *Config {
	b, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}
	config := new(Config)
	err = yaml.Unmarshal(b, config)
	if err != nil {
		log.Fatal(err)
	}

	config.upload_folder = config.FILE_LOCATION + "upload/"
	config.temp_folder = config.FILE_LOCATION + "temp/"
	err = os.MkdirAll(config.upload_folder, os.ModePerm)
	err = os.MkdirAll(config.temp_folder, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	return config

}
func NewTcpServer(configPath string, port int) *TcpServer {
	server := new(TcpServer)
	server.config = readConfig(configPath)
	server.clients = []*client{}
	server.listenPort = port
	server.connect = make(chan *session.Session)
	server.disconnect = make(chan *session.Session)
	return server
}
func NewFileSyncWebServer(configPath string, skip_tls_verify bool) *FileSyncWebServer {
	s := new(FileSyncWebServer)
	s.skip_tls_verify = skip_tls_verify
	s.httpClient = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: skip_tls_verify}}}
	http.DefaultClient = s.httpClient
	s.rac = common.NewRestApiClient(skip_tls_verify)
	s.config = readConfig(configPath)
	s.engine = goweb.Default()
	s.engine.WM.HandlerWidget = &HandlerWidget{s: s}
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

func (s *TcpServer) Serve() {
	// Listen on TCP port 2000 on all available unicast and
	// anycast IP addresses of the local system.
	l, err := net.Listen("tcp", ":"+strconv.Itoa(s.listenPort))
	log.Println("accepting tcp connections on port", s.listenPort)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	// Handle the sessions in a new goroutine.
	go s.serveSessions()
	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		s.connect <- session.NewSession(conn)
	}
}

func (s *TcpServer) serveSessions() {
	for {
		select {
		case connect := <-s.connect:
			client := &client{session: connect, class: 1}
			s.clients = append(s.clients, client)
			s.serveClient(client)
		case disconect := <-s.disconnect:
			disconect.Close()
			for index, item := range s.clients {
				if item.session == disconect {
					s.clients = append(s.clients[:index], s.clients[index+1:]...)
					break
				}
			}
		case _ = <-time.After(time.Second * 1):
			for _, client := range s.clients {
				msg := message.NewMessage(message.MT_SYNC)
				storage := storage.NewSQLManager(s.config.DB_CONN_INFO)
				msg.Header["max"] = -1
				storage.Commit()
				if err := client.session.Send(msg, nil); err != nil {
					go func() {
						s.disconnect <- client.session
					}()
				} else {
					log.Println("notified a client")
				}
			}
		}
	}
}
func (s *TcpServer) serveClient(client *client) {

}
func (s *FileSyncWebServer) Serve() {
	s.bindHandlers(s.engine.RouterGroup.Group())
	apiGroup := s.engine.RouterGroup.Group()
	s.bindApiHandlers(apiGroup)
	addr := s.config.Listen_ip + ":2002"
	log.Println("listening on https://" + addr)
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

func (s *FileSyncWebServer) GetLoginUser(ctx *goweb.Context) (*models.User, error) {
	if ctx.Data["user"] == nil {
		return nil, errors.New("no logged user")
	}
	return ctx.Data["user"].(*models.User), nil
}

type HandlerWidget struct {
	s *FileSyncWebServer
}

func (hw *HandlerWidget) Pre_Process(ctx *goweb.Context) {
	log.Println(ctx.Request.Method, ctx.Request.URL)
}
func (hw *HandlerWidget) Post_Process(ctx *goweb.Context) {
	if ctx.Err != nil {
		accept := ctx.Request.Header.Get("Accept")
		if strings.Contains(accept, "application/json") {
			ctx.Failed(ctx.Err.Error())
		} else {
			data := struct {
				Desc string
			}{Desc: ctx.Err.Error()}
			model := hw.s.newPageModel(ctx, data)
			model.PageTitle = "ERROR"
			ctx.RenderPage(model, "templates/layout.html", "templates/error.html")
		}
	}

	m := ctx.Data["storage"]
	if m != nil {
		if ctx.Ok {
			err := m.(storage.Storage).Commit()
			if err != nil {
				log.Println(err)
			}
		} else {
			m.(storage.Storage).Rollback()
		}
	}

}

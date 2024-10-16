package server

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/swishcloud/gostudy/common"
	"github.com/swishcloud/goweb/auth"

	"github.com/swishcloud/filesync/message"

	"golang.org/x/oauth2"

	"gopkg.in/yaml.v2"

	"github.com/swishcloud/filesync-web/storage"
	"github.com/swishcloud/filesync-web/storage/models"
	"github.com/swishcloud/filesync/session"
	"github.com/swishcloud/goweb"
)

type Config struct {
	Listen_ip      string      `yaml:"listen_ip"`
	Listen_port    string      `yaml:"listen_port"`
	Tcp_port       int         `yaml:"tcp_port"`
	Website_domain string      `yaml:"website_domain"`
	FILE_LOCATION  string      `yaml:"file_location"`
	DB_CONN_INFO   string      `yaml:"db_conn_info"`
	OAuth          ConfigOAuth `yaml:"oauth"`
	upload_folder  string
	temp_folder    string
	Tls_cert_file  string              `yaml:"tls_cert_file"`
	Tls_key_file   string              `yaml:"tls_key_file"`
	HISTORY_DAYS_N int                 `yaml:"HISTORY_DAYS_N"`
	FILESYNC_PATH  string              `yaml:"FILESYNC_PATH"`
	ContentTypes   []ConfigContentType `yaml:"CONTENT_TYPES"`
	CORS_Whitelist []string            `yaml:"CORS_Whitelist"`
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
type ConfigContentType struct {
	Extenstion  string `yaml:"Extenstion"`
	ContentType string `yaml:"ContentType"`
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
	fs         *FileSyncWebServer
	listenPort int
	clients    []*client
	connect    chan *session.Session
	disconnect chan *session.Session
	config     *Config
}
type client struct {
	session      *session.Session
	class        int
	partition_id string
	user         *models.User
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
func NewTcpServer(configPath string, fs *FileSyncWebServer) *TcpServer {
	server := new(TcpServer)
	server.fs = fs
	server.config = readConfig(configPath)
	server.clients = []*client{}
	server.listenPort = server.config.Tcp_port
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
func (s *TcpServer) DeleteHistories() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("DeleteHistories panic:", r)
		}
	}()
	m := storage.NewSQLManager(s.config.DB_CONN_INFO)
	defer func() {
		if err := m.Commit(); err != nil {
			log.Print(err)
		}
	}()
	m.Delete_histories(s.config.HISTORY_DAYS_N)
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
	//run timed tasks
	go func() {
		for {
			s.DeleteHistories()
			time.Sleep(time.Minute)
		}
	}()
	// Handle the sessions in a new goroutine.
	go s.serveSessions()
	for {
		// Wait for a connection.
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		session := session.NewSession(conn)
		s.connect <- session
	}
}
func (s *TcpServer) serveSessions() {
	for {
		select {
		case connect := <-s.connect:
			client := &client{session: connect, class: 1}
			s.clients = append(s.clients, client)
			go s.serveClient(client)
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
				if client.user == nil {
					continue
				}
				msg := message.NewMessage(message.MT_SYNC)
				storage := storage.NewSQLManager(s.config.DB_CONN_INFO)
				defer storage.Commit()
				msg.Header["max"] = -1
				msg.Header["max_commit_id"] = storage.GetPartitionLatestCommit(client.partition_id)["id"]
				msg.Header["first_commit_id"] = storage.GetPartitionFirstCommit(client.partition_id)["id"]
				msg.Header["partition_id"] = client.partition_id
				if err := client.session.Send(msg, nil); err != nil {
					go func(session *session.Session) {
						s.disconnect <- session
					}(client.session)
				} else {
					log.Println("notified a client")
				}
			}
		}
	}
}
func (s *TcpServer) serveClient(client *client) {
	for {
		msg, err := client.session.ReadMessage()
		if err != nil {
			log.Println(err)
			s.disconnect <- client.session
			return
		}
		token := msg.Header["token"]
		if token == nil {
			log.Println("the token is missing")
			s.disconnect <- client.session
			return

		}
		ok, sub, err := auth.CheckToken(s.fs.rac, &oauth2.Token{AccessToken: token.(string)}, s.fs.config.OAuth.IntrospectTokenURL, s.fs.skip_tls_verify)
		if err != nil {
			log.Println(err)
			s.disconnect <- client.session
			return
		}
		if !ok {
			log.Println("the token is invalid:", token)
			s.disconnect <- client.session
			return
		}
		store := storage.NewSQLManager(s.fs.config.DB_CONN_INFO)
		defer store.Commit()
		user := store.GetUserByOpId(sub)
		if user == nil {
			log.Println("not found the user")
			s.disconnect <- client.session
			return
		}
		client.user = user
		client.partition_id = user.Partition_id
	}
}
func (s *FileSyncWebServer) Serve() {

	s.bindHandlers(s.engine.RouterGroup.Group())
	apiGroup := s.engine.RouterGroup.Group()
	s.bindApiHandlers(apiGroup)
	addr := s.config.Listen_ip + ":" + s.config.Listen_port
	log.Println("Website Url: https://" + s.config.Website_domain)
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
func (s *FileSyncWebServer) showErrorPage(ctx *goweb.Context, status int, msg string) {
	data := struct {
		Desc string
	}{Desc: msg}
	model := s.newPageModel(ctx, data)
	model.PageTitle = "ERROR"
	ctx.Writer.WriteHeader(status)
	ctx.RenderPage(model, "templates/layout.html", "templates/error.html")
}

type HandlerWidget struct {
	s *FileSyncWebServer
}

func (hw *HandlerWidget) Pre_Process(ctx *goweb.Context) {
	log.Println(ctx.Request.Method, ctx.Request.URL)
}
func (hw *HandlerWidget) Post_Process(ctx *goweb.Context) {
	m := ctx.Data["storage"]
	if m != nil {
		err := m.(storage.Storage).Commit()
		if err != nil {
			log.Println(err)
		}
	}
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
}

package server

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type (
	Server struct {
		ENV  string `json:"env,omitempty"`
		Name string `json:"name,omitempty"`

		Addr              string        `json:"addr,omitempty"`
		Certificates      []Certificate `json:"certificates,omitempty"`
		ReadTimeout       time.Duration `json:"read_timeout,omitempty"`
		ReadHeaderTimeout time.Duration `json:"read_header_timeout,omitempty"`
		WriteTimeout      time.Duration `json:"write_timeout,omitempty"`
		IdleTimeout       time.Duration `json:"idle_timeout,omitempty"`
		ShutdownTimeout   time.Duration `json:"shutdown_timeout,omitempty"`

		Compress *Compress  `json:"compress,omitempty"`
		Logger   *Logger    `json:"logger,omitempty"`
		Redis    *Redis     `json:"redis,omitempty"`
		Mongo    *Mongo     `json:"mongo,omitempty"`
		Handlers []*Handler `json:"handlers,omitempty"`
	}
)

func (server *Server) Init() *Server {
	switch server.ENV {
	case "dev", "development":
		server.ENV = "development"
	case "test":
		server.ENV = "test"
	default:
		server.ENV = "production"
	}

	if server.Name == "" {
		dir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		dir, server.Name = path.Split(strings.Trim(dir, "/\\"))
		if server.Name == "" {
			server.Name = "unnamed"
		}
		server.Name = strings.ToLower(server.Name)
	}

	if server.Addr == "" {
		if server.Certificates == nil {
			server.Addr = ":8080"
		} else {
			server.Addr = ":8443"
		}
	}

	if strings.HasSuffix(server.Addr, ":443") || strings.HasSuffix(server.Addr, ":8443") || (server.Certificates != nil && len(server.Certificates) == 0) {
		if len(server.Certificates) == 0 {
			priv, cert, err := NewCertificate("localhost", []string{"localhost"}, "ecdsa", 384)
			if err != nil {
				panic(err)
			}

			certificate, err2 := EncodeCertificate(priv, cert)
			if err2 != nil {
				panic(err)
			}

			server.Certificates = append(server.Certificates, certificate)
		}
	}

	if server.ReadTimeout == 0 {
		server.ReadTimeout = time.Second * 20
	}
	if server.ReadHeaderTimeout == 0 {
		server.ReadHeaderTimeout = time.Second * 10
	}
	if server.WriteTimeout == 0 {
		server.WriteTimeout = time.Second * 30
	}
	if server.IdleTimeout == 0 {
		server.IdleTimeout = time.Second * 300
	}
	if server.ShutdownTimeout == 0 {
		server.ShutdownTimeout = server.WriteTimeout + server.ReadTimeout + server.ReadHeaderTimeout
	}

	// gin
	switch server.ENV {
	case "development":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	if server.Compress == nil {
		server.Compress = &Compress{}
	}
	if server.Logger == nil {
		server.Logger = &Logger{}
	}
	if server.Redis == nil {
		server.Redis = &Redis{}
	}
	if server.Mongo == nil {
		server.Mongo = &Mongo{}
	}
	server.Compress.init(server, nil)
	server.Logger.init(server, nil)
	server.Redis.init(server, nil)
	server.Mongo.init(server, nil)

	return server
}

func (server *Server) Get(name string, create bool) (handler *Handler) {
	for _, val := range server.Handlers {
		if val.Name == name {
			handler = val
			break
		}
	}
	if handler == nil && create {
		handler = &Handler{
			Name: name,
		}
		server.Handlers = append(server.Handlers, handler)
	}
	if handler != nil {
		handler.Init(server)
	}
	return
}

func (server *Server) Start() {
	var tlsConfig *tls.Config
	if len(server.Certificates) != 0 {
		var certificates []tls.Certificate
		for _, val := range server.Certificates {
			certificate, err := tls.X509KeyPair([]byte(val.Certificate), []byte(val.PrivateKey))
			if err != nil {
				panic(err)
			}
			certificates = append(certificates, certificate)
		}
		tlsConfig = &tls.Config{
			MinVersion:               tls.VersionTLS10,
			Certificates:             certificates,
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			},
		}
		tlsConfig.BuildNameToCertificate()
	}

	logWriter := server.Logger.Get().Writer()
	defer logWriter.Close()

	handler := serverHandler{}
	for _, val := range server.Handlers {
		for _, host := range val.Hosts {
			if val.Get() != nil {
				handler[host] = val.Get()
			}
		}
	}

	httpServer := http.Server{
		Addr:              server.Addr,
		Handler:           handler,
		TLSConfig:         tlsConfig,
		ReadTimeout:       server.ReadTimeout,
		ReadHeaderTimeout: server.ReadHeaderTimeout,
		WriteTimeout:      server.WriteTimeout,
		IdleTimeout:       server.IdleTimeout,
		MaxHeaderBytes:    4096,
		ErrorLog:          log.New(logWriter, "", 0),
	}

	// 执行
	go func() {
		var err error
		if tlsConfig == nil {
			err = httpServer.ListenAndServe()
		} else {
			err = httpServer.ListenAndServeTLS("", "")
		}
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutdown Server ...")
	//
	ctx, cancel := context.WithTimeout(context.Background(), server.ShutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Panic("Server Shutdown:", err)
	}

	log.Println("Server exiting")
}

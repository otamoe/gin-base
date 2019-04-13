package engine

import (
	"compress/gzip"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/go-redis/redis"
	"github.com/otamoe/gin-engine/compress"
	"github.com/otamoe/gin-engine/errors"
	"github.com/otamoe/gin-engine/logger"
	"github.com/otamoe/gin-engine/mongo"
	"github.com/otamoe/gin-engine/notfound"
	ginRedis "github.com/otamoe/gin-engine/redis"
	"github.com/otamoe/gin-engine/size"
	"github.com/sirupsen/logrus"
)

type (
	MongoConfig struct {
		URL         string
		DialTimeout time.Duration
	}

	LoggerConfig struct {
		File string
	}

	ServerConfig struct {
		Addr              string
		Certificates      []tls.Certificate
		ReadTimeout       time.Duration
		ReadHeaderTimeout time.Duration
		WriteTimeout      time.Duration
		IdleTimeout       time.Duration
		MaxHeaderBytes    int
	}

	Handler map[string]http.Handler

	Engine struct {
		ENV  string
		Name string

		LoggerConfig   LoggerConfig
		MongoConfig    MongoConfig
		RedisConfig    redis.Options
		CompressConfig compress.Config
		ServerConfig   ServerConfig

		Handler Handler

		mongoSession *mgo.Session
	}
)

func (engine *Engine) Init() *Engine {
	switch engine.ENV {
	case "dev":
		engine.ENV = "development"
	case "prod", "":
		engine.ENV = "production"
	}
	if engine.Name == "" {
		dir, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		dir, engine.Name = path.Split(strings.Trim(dir, "/\\"))
		if engine.Name == "" {
			engine.Name = "unnamed"
		}
		engine.Name = strings.ToLower(engine.Name)
	}
	engine.initGin()
	engine.initCompress()
	engine.initLogger()
	engine.initRedis()
	engine.initMongo()

	return engine
}
func (engine *Engine) initGin() {
	switch engine.ENV {
	case "development":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}
}

func (engine *Engine) initCompress() {
	config := engine.CompressConfig
	if config.GzipLevel == 0 {
		config.GzipLevel = gzip.DefaultCompression
	}
	if config.MinLength == 0 {
		config.MinLength = 256
	}
	if config.BrLGWin == 0 {
		config.BrLGWin = 19
	}
	if config.BrQuality == 0 {
		config.BrQuality = 6
	}
	if config.Types == nil {
		config.Types = []string{"application/json", "text/plain"}
	}

	engine.CompressConfig = config
}

func (engine *Engine) initLogger() {
	switch engine.ENV {
	case "development":
		logrus.SetLevel(logrus.TraceLevel)
	case "test":
		logrus.SetLevel(logrus.TraceLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})

	logrus.SetOutput(os.Stdout)
	if engine.LoggerConfig.File != "" {
		writer, err := os.OpenFile(engine.LoggerConfig.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
		logrus.SetOutput(writer)
	}
	log.SetOutput(logrus.StandardLogger().Writer())
}

func (engine *Engine) initRedis() {
	config := engine.RedisConfig
	if config.Addr == "" {
		config.Addr = "localhost:6379"
	}
	if config.PoolSize == 0 {
		config.PoolSize = 4096
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.DialTimeout == 0 {
		config.DialTimeout = time.Second * 2
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = time.Second * 2
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = time.Second * 2
	}
	if config.PoolTimeout == 0 {
		config.PoolTimeout = time.Second * 2
	}
	engine.RedisConfig = config

	logWriter := engine.Logger().Writer()
	redis.SetLogger(log.New(logWriter, "", 0))
}

func (engine *Engine) initMongo() {
	config := engine.MongoConfig
	if config.URL == "" {
		config.URL = "localhost:27017/" + engine.Name
	}
	if config.DialTimeout == 0 {
		config.DialTimeout = time.Second * 2
	}
	engine.MongoConfig = config

	if engine.ENV == "development" {
		mgo.SetDebug(true)
	}

	logWriter := engine.Logger().Writer()
	mgo.SetLogger(log.New(logWriter, "", 0))

	var err error
	engine.mongoSession, err = mgo.DialWithTimeout(config.URL, config.DialTimeout)
	if err != nil {
		panic(err)
	}
	engine.mongoSession.SetPoolLimit(4096)
}

func (engine *Engine) initServer() {
	config := engine.ServerConfig
	if config.Addr == "" {
		if config.Certificates == nil {
			config.Addr = ":8080"
		} else {
			config.Addr = ":8443"
		}
	}
	if strings.HasSuffix(config.Addr, ":443") || strings.HasSuffix(config.Addr, ":8443") || (config.Certificates != nil && len(config.Certificates) == 0) {
		if len(config.Certificates) == 0 {
			for host := range engine.Handler {
				priv, cert, err := NewCertificate(host, []string{host}, "ecdsa", 384)
				if err != nil {
					panic(err)
				}
				config.Certificates = append(config.Certificates, tls.Certificate{
					Certificate: [][]byte{cert},
					PrivateKey:  priv,
				})
			}
		}
	}

	if config.ReadTimeout == 0 {
		config.ReadTimeout = time.Second * 20
	}
	if config.ReadHeaderTimeout == 0 {
		config.ReadHeaderTimeout = time.Second * 10
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = time.Second * 30
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = time.Second * 300
	}
	if config.MaxHeaderBytes == 0 {
		config.MaxHeaderBytes = 4096
	}

	engine.ServerConfig = config
}

func (engine *Engine) Logger() *logrus.Logger {
	return logrus.StandardLogger()
}

func (engine *Engine) Redis() (client *redis.Client) {
	client = redis.NewClient(&engine.RedisConfig)
	return
}

func (engine *Engine) Mongo() (session *mgo.Session) {
	session = engine.mongoSession.Clone()
	return
}

func (engine *Engine) New() (r *gin.Engine) {
	r = gin.New()

	// Compress 中间件
	r.Use(compress.Middleware(engine.CompressConfig))

	// Redis 中间件
	r.Use(ginRedis.Middleware(engine.Redis))

	// Mongo 中间件
	r.Use(mongo.Middleware(engine.Mongo))

	// logger
	r.Use(logger.Middleware(logger.Config{
		Prefix: "[HTTP] ",
	}))

	// errors
	r.Use(errors.Middleware())

	// body size
	r.Use(size.Middleware(1024 * 512))

	// 未匹配
	r.NoRoute(notfound.Middleware())

	return
}

func (engine *Engine) Server() {
	engine.initServer()
	config := engine.ServerConfig
	var tlsConfig *tls.Config
	if config.Certificates != nil {
		tlsConfig = &tls.Config{
			MinVersion:               tls.VersionTLS10,
			Certificates:             config.Certificates,
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

	logWriter := engine.Logger().Writer()
	defer logWriter.Close()

	server := http.Server{
		Addr:              config.Addr,
		Handler:           engine.Handler,
		TLSConfig:         tlsConfig,
		ReadTimeout:       config.ReadTimeout,
		ReadHeaderTimeout: config.ReadHeaderTimeout,
		WriteTimeout:      config.WriteTimeout,
		IdleTimeout:       config.IdleTimeout,
		MaxHeaderBytes:    config.MaxHeaderBytes,
		ErrorLog:          log.New(logWriter, "", 0),
	}

	// 执行
	go func() {
		var err error
		if tlsConfig == nil {
			err = server.ListenAndServe()
		} else {
			err = server.ListenAndServeTLS("", "")
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
	ctx, cancel := context.WithTimeout(context.Background(), config.ReadTimeout+config.WriteTimeout+config.ReadHeaderTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Panic("Server Shutdown:", err)
	}

	log.Println("Server exiting")
}

func (h Handler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/favicon.ico":
		writer.Header().Set("Content-Type", "image/x-icon")
		writer.WriteHeader(http.StatusOK)
		fmt.Fprintln(writer, "")
	case "/robots.txt":
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		fmt.Fprintln(writer, "Disallow: /")
	case "/crossdomain.xml":
		writer.Header().Set("Content-Type", "application/xml; charset=utf-8")
		writer.WriteHeader(http.StatusOK)
		fmt.Fprintln(writer, "<?xml version=\"1.0\"?><cross-domain-policy></cross-domain-policy>")
	default:
		var host string
		if host = req.Header.Get("X-Forwarded-Host"); host != "" {
		} else if host = req.Header.Get("X-Host"); host != "" {
		} else if host = req.Host; host != "" {
		} else if host = req.URL.Host; host != "" {
		} else {
			host = "localhost"
		}

		if host != "" {
			if index := strings.LastIndex(host, ":"); index != -1 {
				host = host[0:index]
			}
		}

		if mux := h[host]; mux != nil {
			mux.ServeHTTP(writer, req)
		} else if mux := h["default"]; mux != nil {
			mux.ServeHTTP(writer, req)
		} else {
			http.Error(writer, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	}
}
func NewCertificate(name string, hosts []string, typ string, bits int) (priv crypto.PrivateKey, cert []byte, err error) {
	var pub crypto.PublicKey
	switch typ {
	case "ecdsa":
		{
			var privateKey *ecdsa.PrivateKey
			switch bits {
			case 224:
				privateKey, err = ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
			case 256:
				privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			case 384:
				privateKey, err = ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
			case 521:
				privateKey, err = ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
			}
			if err != nil {
				return
			}
			priv = privateKey
			pub = privateKey.Public()
		}
	default:
		{
			var privateKey *rsa.PrivateKey
			if privateKey, err = rsa.GenerateKey(rand.Reader, bits); err != nil {
				return
			}
			priv = privateKey
			pub = privateKey.Public()
		}
	}

	max := new(big.Int).Lsh(big.NewInt(1), 128)
	var serialNumber *big.Int
	if serialNumber, err = rand.Int(rand.Reader, max); err != nil {
		return
	}

	subject := pkix.Name{
		Organization:       []string{"Organization"},
		OrganizationalUnit: []string{"Organizational Unit"},
		CommonName:         name,
	}

	template := &x509.Certificate{
		SerialNumber:        serialNumber,
		Subject:             subject,
		NotBefore:           time.Now().Add(-(time.Hour * 24 * 30)),
		NotAfter:            time.Now().Add(time.Hour * 24 * 365 * 20),
		KeyUsage:            x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:         []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		PermittedDNSDomains: hosts,
		PermittedURIDomains: hosts,
	}

	if cert, err = x509.CreateCertificate(rand.Reader, template, template, pub, priv); err != nil {
		return
	}

	return
}

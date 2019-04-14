package server

import (
	"compress/gzip"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/otamoe/gin-server/compress"
	"github.com/otamoe/gin-server/errors"
	"github.com/otamoe/gin-server/logger"
	"github.com/otamoe/gin-server/mongo"
	"github.com/otamoe/gin-server/notfound"
	ginRedis "github.com/otamoe/gin-server/redis"
	"github.com/otamoe/gin-server/size"
)

type (
	Handler struct {
		Name     string    `json:"name,omitempty"`
		Hosts    []string  `json:"hosts,omitempty"`
		Compress *Compress `json:"compress,omitempty"`
		Logger   *Logger   `json:"logger,omitempty"`
		Redis    *Redis    `json:"redis,omitempty"`
		Mongo    *Mongo    `json:"mongo,omitempty"`
		gin      *gin.Engine
	}

	serverHandler map[string]*gin.Engine
)

func (handler *Handler) Init(server *Server) {
	if handler.gin != nil {
		return
	}

	if handler.Compress == nil {
		handler.Compress = server.Compress
	} else {
		handler.Compress.init(server, handler)
	}
	if handler.Logger == nil {
		handler.Logger = server.Logger
	} else {
		handler.Logger.init(server, handler)
	}
	if handler.Redis == nil {
		handler.Redis = server.Redis
	} else {
		handler.Redis.init(server, handler)
	}
	if handler.Mongo == nil {
		handler.Mongo = server.Mongo
	} else {
		handler.Mongo.init(server, handler)
	}

	handler.gin = gin.New()

	// Compress 中间件
	handler.gin.Use(compress.Middleware(compress.Config{
		GzipLevel: gzip.DefaultCompression,
		MinLength: 256,
		BrLGWin:   19,
		BrQuality: 6,
		Types:     handler.Compress.Types,
	}))

	// logger
	handler.gin.Use(logger.Middleware(logger.Config{
		Prefix: "[HTTP] ",
		Logger: handler.Logger.Get(),
	}))

	// errors
	handler.gin.Use(errors.Middleware())

	// Redis 中间件
	handler.gin.Use(ginRedis.Middleware(handler.Redis.Get))

	// Mongo 中间件
	handler.gin.Use(mongo.Middleware(handler.Mongo.Get))

	// body size
	handler.gin.Use(size.Middleware(1024 * 512))

	// 未匹配
	handler.gin.NoRoute(notfound.Middleware())

}

func (h serverHandler) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
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

		if mux, ok := h[host]; ok {
			mux.ServeHTTP(writer, req)
		} else if mux, ok := h["default"]; ok {
			mux.ServeHTTP(writer, req)
		} else {
			http.Error(writer, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		}
	}
}

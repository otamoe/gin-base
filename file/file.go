package file

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/otamoe/gin-server/errs"
	"github.com/otamoe/gin-server/logger"
)

type (
	Config struct {
		Root    string
		Control []string
		Logger  bool
	}
)

func filtered(val string) bool {
	for _, name := range strings.FieldsFunc(val, isSlashRune) {
		name = strings.TrimSpace(name)
		if name == "" {
			return true
		}
		if name[0] == '.' {
			return true
		}
	}
	return false
}

func isSlashRune(r rune) bool {
	return r == '/' || r == '\\'
}

func Middleware(c Config) gin.HandlerFunc {
	fileserver := http.FileServer(http.Dir(c.Root))
	return func(ctx *gin.Context) {
		urlPath := ctx.Request.URL.Path
		if !strings.HasPrefix(urlPath, "/") {
			urlPath = "/" + urlPath
		}

		if filtered(urlPath) {
			ctx.Abort()
			ctx.Error(&errs.Error{
				Message:    http.StatusText(http.StatusBadRequest),
				StatusCode: http.StatusBadRequest,
			})
			return
		}

		if ctx.Request.Method != http.MethodGet && ctx.Request.Method != http.MethodHead {
			ctx.Next()
			return
		}

		name := path.Join(c.Root, urlPath)
		stats, err := os.Stat(name)
		if err != nil || stats.IsDir() {
			ctx.Next()
			return
		}

		if !c.Logger {
			ctx.Set(logger.CONTEXT, nil)
		}

		ctx.Header("cache-control", strings.Join(c.Control, ","))
		ctx.Header("etag", "\""+fmt.Sprint(stats.ModTime().Unix())+"\"")

		fileserver.ServeHTTP(ctx.Writer, ctx.Request)
		ctx.Abort()
	}
}

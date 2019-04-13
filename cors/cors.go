package cors

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type (
	Config struct {
		Origins []string
		MaxAge  int
	}
)

func Middleware(c Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if len(c.Origins) != 0 {
			ctx.Header("Access-Control-Allow-Methods", "HEAD, GET, PUT, PATCH, POST, DELETE")
			ctx.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, Range, If-Match, If-Modified-Since, If-None-Match, If-Range, If-Unmodified-Since")
			ctx.Header("Access-Control-Expose-Headers", "Accept-Ranges, Content-Range, Content-Length, Content-Disposition, ETag, Date")

			ctx.Header("Access-Control-Max-Age", strconv.Itoa(c.MaxAge))
			ctx.Header("Access-Control-Allow-Origin", strings.Join(c.Origins, ","))
			if ctx.Request.Method == http.MethodOptions {
				ctx.AbortWithStatus(http.StatusOK)
				return
			}
		}
		ctx.Next()
	}
}

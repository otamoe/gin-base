package notfound

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/otamoe/gin-server/errs"
)

func Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.AbortWithError(http.StatusNotFound, &errs.Error{
			Message:    http.StatusText(http.StatusNotFound),
			Type:       "not_found",
			StatusCode: http.StatusNotFound,
		})
	}
}

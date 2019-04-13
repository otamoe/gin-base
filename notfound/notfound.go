package notfound

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/otamoe/gin-engine/errors"
)

func Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.AbortWithError(http.StatusNotFound, &errors.Error{
			Message:    http.StatusText(http.StatusNotFound),
			Type:       "not_found",
			StatusCode: http.StatusNotFound,
		})
	}
}

package abort

import "github.com/gin-gonic/gin"

func Middleware(status int) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if status == 0 {
			ctx.Abort()
		} else {
			ctx.AbortWithStatus(status)
		}
	}
}

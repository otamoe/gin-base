package resource

import (
	"github.com/gin-gonic/gin"
)

var CONTEXT_HANDLER = "GIN.SERVER.RESOURCE.HANDLER"
var CONTEXT_TYPE = "GIN.SERVER.RESOURCE.TYPE"
var CONTEXT_ACTION = "GIN.SERVER.RESOURCE.ACTION"

func Handler(handler string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set(CONTEXT_HANDLER, handler)
		ctx.Next()
	}
}

func Middleware(typ, action string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set(CONTEXT_TYPE, typ)
		ctx.Set(CONTEXT_ACTION, action)
		ctx.Next()
	}
}

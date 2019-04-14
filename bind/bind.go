package bind

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	DataFunc func(ctx *gin.Context) interface{}
)

var CONTEXT = "GIN.ENGINE.BIND"

func Middleware(dataFunc DataFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err error
		data := dataFunc(ctx)
		if err = ctx.ShouldBind(data); err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err).SetType(gin.ErrorTypeBind)
			return
		}
		ctx.Set(CONTEXT, data)
		ctx.Next()
	}
}

func MiddlewareQuery(dataFunc DataFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err error
		data := dataFunc(ctx)
		if err = ctx.ShouldBindQuery(data); err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err).SetType(gin.ErrorTypeBind)
			return
		}
		ctx.Set(CONTEXT, data)
		ctx.Next()
	}
}

func MiddlewareJSON(dataFunc DataFunc) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err error
		data := dataFunc(ctx)
		if err = ctx.ShouldBindJSON(data); err != nil {
			ctx.AbortWithError(http.StatusBadRequest, err).SetType(gin.ErrorTypeBind)
			return
		}
		ctx.Set(CONTEXT, data)
		ctx.Next()
	}
}

package scope

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/otamoe/gin-server/errs"
	ginResource "github.com/otamoe/gin-server/resource"
)

type (
	Interface interface {
		ValidateScope(resource *ginResource.Resource) (params map[string]interface{}, err error)
	}
)

var CONTEXT = "GIN.SERVER.SCOPE"
var CONTEXT_PARAMS = "GIN.SERVER.SCOPE.PARAMS"
var CONTEXT_ERROR = "GIN.SERVER.SCOPE.ERROR"
var ErrRequired = &errs.Error{
	Message:    "You are not logged in",
	Type:       "token",
	StatusCode: http.StatusUnauthorized,
}

func Middleware(required bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err error
		var params map[string]interface{}
		if val, ok := ctx.Get(CONTEXT); ok {
			resource := ctx.MustGet(ginResource.CONTEXT).(*ginResource.Resource)
			resource.Pre()
			params, err = val.(Interface).ValidateScope(resource)
		} else {
			err = ErrRequired
		}
		if params == nil {
			params = map[string]interface{}{}
		}
		ctx.Set(CONTEXT_PARAMS, params)
		ctx.Set(CONTEXT_ERROR, err)
		if err != nil && required {
			ctx.Error(err)
			ctx.Abort()
		} else {
			ctx.Next()
		}
	}
}

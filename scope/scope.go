package scope

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/otamoe/gin-server/errs"
	ginResource "github.com/otamoe/gin-server/resource"
)

type (
	ScopeInterface interface {
		ValidateScope(resource *ginResource.Resource) (params map[string]interface{}, err error)
	}
)

var CONTEXT = "GIN.SERVER.SCOPE"
var CONTEXT_PARAMS = "GIN.SERVER.SCOPE.PARAMS"
var CONTEXT_ERROR = "GIN.SERVER.SCOPE.ERROR"

func Middleware(required bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err error
		var params map[string]interface{}
		resource := ctx.MustGet(ginResource.CONTEXT).(*ginResource.Resource)
		if val, ok := ctx.Get(CONTEXT); ok {
			params, err = val.(ScopeInterface).ValidateScope(resource)
		} else {
			errParams := map[string]interface{}{
				"handler": resource.Handler,
				"type":    resource.Type,
				"action":  resource.Action,
				"value":   resource.GetValue(),
			}
			for name, val := range resource.Params {
				errParams[name] = val
			}
			err = &errs.Error{
				Message:    "Validate Scope",
				Type:       "scope",
				StatusCode: http.StatusForbidden,
				Params:     errParams,
			}
		}
		if params == nil {
			params = map[string]interface{}{}
		}
		ctx.Set(CONTEXT_PARAMS, params)
		ctx.Set(CONTEXT_ERROR, err)
		ctx.Next()
	}
}

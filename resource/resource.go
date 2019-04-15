package resource

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/otamoe/gin-server/utils"
)

type (
	Config struct {
		Handler   string
		Type      string
		Action    string
		Params    map[string]interface{}
		ValueKeys []string
		OwnerKeys []string
	}
	Resource struct {
		config  *Config
		context *gin.Context

		Handler string
		Type    string
		Action  string
		Params  map[string]interface{}
	}
)

var CONTEXT = "GIN.SERVER.RESOURCE"
var DEFAULT_HANDLER = "GIN.SERVER.RESOURCE.HANDLER"

func DefaultHandler(handler string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set(DEFAULT_HANDLER, handler)
		ctx.Next()
	}
}

func Middleware(config Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		resource := &Resource{
			config:  &config,
			context: ctx,
			Handler: config.Handler,
			Type:    config.Type,
			Action:  config.Action,
			Params:  map[string]interface{}{},
		}
		if resource.Handler == "" {
			resource.Handler = ctx.GetString(DEFAULT_HANDLER)
		}
		for name, val := range config.Params {
			resource.Params[name] = val
		}
		ctx.Set(CONTEXT, resource)
		ctx.Next()
	}
}

func (resource *Resource) GetValue() (value string) {
	if len(resource.config.ValueKeys) == 0 {
		return
	}
	if val, ok := utils.GetContextValue(resource.context, resource.config.ValueKeys); ok && val != nil {
		switch val := val.(type) {
		case bson.ObjectId:
			value = val.Hex()
		case fmt.Stringer:
			value = val.String()
		default:
			value = fmt.Sprintf("%+v", val)
		}
	}
	return
}

func (resource *Resource) GetOwner() (owner bson.ObjectId) {
	if len(resource.config.OwnerKeys) == 0 {
		return
	}
	if val, ok := utils.GetContextValue(resource.context, resource.config.OwnerKeys); ok && val != nil {
		switch val := val.(type) {
		case bson.ObjectId:
			owner = val
		case fmt.Stringer:
			if bson.IsObjectIdHex(val.String()) {
				owner = bson.ObjectIdHex(val.String())
			}
		default:
			val2 := fmt.Sprintf("%+v", val)
			if bson.IsObjectIdHex(val2) {
				owner = bson.ObjectIdHex(val2)
			}
		}
	}
	return
}

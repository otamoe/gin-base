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
		ValueKeys []string
		OwnerKeys []string
		Params    map[string]interface{}
	}
	Resource struct {
		context *gin.Context

		Handler   string
		Type      string
		Action    string
		ValueKeys []string
		OwnerKeys []string
		Params    map[string]interface{}
	}
)

var CONTEXT = "GIN.SERVER.RESOURCE"

func Middleware(config Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var resource *Resource
		if val, ok := ctx.Get(CONTEXT); ok {
			resource = val.(*Resource)
		} else {
			resource = &Resource{
				context: ctx,
				Params:  map[string]interface{}{},
			}
			ctx.Set(CONTEXT, resource)
		}
		if config.Handler != "" {
			resource.Handler = config.Handler
		}
		if config.Type != "" {
			resource.Type = config.Type
		}
		if config.Action != "" {
			resource.Action = config.Action
		}

		if config.ValueKeys != nil {
			resource.ValueKeys = config.ValueKeys
		}
		if config.OwnerKeys != nil {
			resource.OwnerKeys = config.OwnerKeys
		}

		for key, val := range config.Params {
			resource.Params[key] = val
		}
		ctx.Next()
	}
}

func (resource *Resource) GetValue() (value string) {
	if len(resource.ValueKeys) == 0 {
		return
	}
	if val, ok := utils.GetContextValue(resource.context, resource.ValueKeys); ok && val != nil {
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
	if len(resource.OwnerKeys) == 0 {
		return
	}
	if val, ok := utils.GetContextValue(resource.context, resource.OwnerKeys); ok && val != nil {
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

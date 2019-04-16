package resource

import (
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/otamoe/gin-server/utils"
)

type (
	Config struct {
		Handler   string
		Type      string
		Action    string
		Value     string
		Owner     bson.ObjectId
		ValueKeys []string
		OwnerKeys []string
		Params    map[string]interface{}
	}
	Resource struct {
		context *gin.Context

		Handler   string
		Type      string
		Action    string
		Value     string
		Owner     bson.ObjectId
		ValueKeys []string
		OwnerKeys []string
		Params    map[string]interface{}
	}
)

var CONTEXT = "GIN.SERVER.RESOURCE"

var handlersMap = sync.Map{}

func Handler(handler gin.HandlerFunc, config Config) {
	handlersMap.Store(utils.NameOfFunction(handler), config)
	return
}

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
		resource.Config(config)
		if val, ok := handlersMap.Load(ctx.HandlerName()); ok && val != nil {
			resource.Config(val.(Config))
		}
		ctx.Next()
	}
}

func (resource *Resource) Config(config Config) {
	if config.Handler != "" {
		resource.Handler = config.Handler
	}
	if config.Type != "" {
		resource.Type = config.Type
	}
	if config.Action != "" {
		resource.Action = config.Action
	}
	if config.Value != "" {
		resource.Value = config.Value
	}

	if config.Owner != "" {
		resource.Owner = config.Owner
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
}
func (resource *Resource) GetValue() (value string) {
	if resource.Value != "" {
		value = resource.Value
		return
	}
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
	if resource.Owner != "" {
		owner = resource.Owner
		return
	}
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

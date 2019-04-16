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
		Application bson.ObjectId
		Type        string
		Action      string
		Value       string
		Owner       bson.ObjectId
		ValueKeys   []string
		OwnerKeys   []string
		Params      map[string]interface{}
	}
	Resource struct {
		context   *gin.Context
		ValueKeys []string
		OwnerKeys []string

		Application bson.ObjectId
		Type        string
		Action      string
		Value       string
		Owner       bson.ObjectId
		Params      map[string]interface{}
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
	if config.Application != "" {
		resource.Application = config.Application
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
func (resource *Resource) GetValue() string {
	if resource.Value == "" && len(resource.ValueKeys) != 0 {
		if val, ok := utils.GetContextValue(resource.context, resource.ValueKeys); ok && val != nil {
			switch val := val.(type) {
			case bson.ObjectId:
				resource.Value = val.Hex()
			case fmt.Stringer:
				resource.Value = val.String()
			default:
				resource.Value = fmt.Sprintf("%+v", val)
			}
		}
	}

	return resource.Value
}

func (resource *Resource) GetOwner() bson.ObjectId {
	if resource.Owner == "" && len(resource.OwnerKeys) != 0 {
		if val, ok := utils.GetContextValue(resource.context, resource.OwnerKeys); ok && val != nil {
			switch val := val.(type) {
			case bson.ObjectId:
				resource.Owner = val
			case fmt.Stringer:
				if bson.IsObjectIdHex(val.String()) {
					resource.Owner = bson.ObjectIdHex(val.String())
				}
			default:
				val2 := fmt.Sprintf("%+v", val)
				if bson.IsObjectIdHex(val2) {
					resource.Owner = bson.ObjectIdHex(val2)
				}
			}
		}
	}
	return resource.Owner
}

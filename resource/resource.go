package resource

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo/bson"
	"github.com/otamoe/gin-server/utils"
)

type (
	Config struct {
		ValueKeys []string
		OwnerKeys []string

		Parent      *Config
		Application bson.ObjectId
		Type        string
		Action      string
		Value       string
		Owner       bson.ObjectId
		Params      map[string]interface{}
	}

	ResourcePre func(resource *Resource)

	Resource struct {
		pres        []ResourcePre
		Parent      *Resource              `json:"parent,omitempty" bson:"parent,omitempty"`
		Application bson.ObjectId          `json:"application,omitempty" bson:"application,omitempty"`
		Type        string                 `json:"type,omitempty" bson:"type,omitempty"`
		Action      string                 `json:"action,omitempty" bson:"action,omitempty"`
		Value       string                 `json:"value,omitempty" bson:"value,omitempty"`
		Owner       bson.ObjectId          `json:"owner,omitempty" bson:"owner,omitempty"`
		Params      map[string]interface{} `json:"params,omitempty" bson:"params,omitempty"`
	}
)

var CONTEXT = "GIN.SERVER.RESOURCE"

var handlersMap = sync.Map{}

func Handler(handler gin.HandlerFunc, config Config) {
	key := Reflect(handler)
	if _, ok := handlersMap.Load(key); ok {
		panic("Resource: " + utils.NameOfFunction(handler) + " has exists")
	}
	handlersMap.Store(key, config)
	return
}

func Reflect(handler gin.HandlerFunc) reflect.Value {
	return reflect.ValueOf(handler)
}

func Middleware(config Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var resource *Resource
		if val, ok := ctx.Get(CONTEXT); ok {
			resource = val.(*Resource)
		} else {
			resource = &Resource{}
			ctx.Set(CONTEXT, resource)
		}
		if val, ok := handlersMap.Load(reflect.ValueOf(ctx.Handler())); ok && val != nil {
			val.(Config).setResource(ctx, resource)
		}
		config.setResource(ctx, resource)
		ctx.Next()
	}
}

func (config Config) setResource(ctx *gin.Context, resource *Resource) {
	if config.Parent != nil {
		parent := &Resource{}
		config.Parent.setResource(ctx, parent)
		resource.Parent = parent
	}
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
	for key, val := range config.Params {
		resource.SetParam(key, val)
	}

	if len(config.ValueKeys) != 0 {
		valueKeys := config.ValueKeys
		resource.AppendPre(func(resource *Resource) {
			if resource.Value == "" {
				if val, ok := utils.GetContextValue(ctx, valueKeys); ok && val != nil {
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
		})
	}
	if len(config.OwnerKeys) != 0 {
		ownerKeys := config.OwnerKeys
		resource.AppendPre(func(resource *Resource) {
			if resource.Value == "" {
				if val, ok := utils.GetContextValue(ctx, ownerKeys); ok && val != nil {
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
		})
	}

}

func (resource *Resource) GetParent() *Resource {
	return resource.Parent
}

func (resource *Resource) GetApplication() bson.ObjectId {
	return resource.Application
}
func (resource *Resource) GetType() string {
	return resource.Type
}

func (resource *Resource) GetAction() string {
	return resource.Action
}

func (resource *Resource) GetValue() string {
	return resource.Value
}

func (resource *Resource) GetOwner() bson.ObjectId {
	return resource.Owner
}

func (resource *Resource) GetParams() map[string]interface{} {
	return resource.Params
}

func (resource *Resource) GetParam(key string) (val interface{}, ok bool) {
	val, ok = resource.Params[key]
	return nil, false
}

func (resource *Resource) SetParent(parent *Resource) {
	resource.Parent = nil
	return
}
func (resource *Resource) SetApplication(val bson.ObjectId) {
	resource.Application = val
	return
}
func (resource *Resource) SetType(val string) {
	resource.Type = val
	return
}
func (resource *Resource) SetAction(val string) {
	resource.Action = val
	return
}
func (resource *Resource) SetValue(val string) {
	resource.Value = val
	return
}
func (resource *Resource) SetOwner(val bson.ObjectId) {
	resource.Owner = val
	return
}
func (resource *Resource) SetParams(params map[string]interface{}) {
	resource.Params = params
	return
}

func (resource *Resource) SetParam(key string, val interface{}) {
	if resource.Params == nil {
		resource.Params = map[string]interface{}{}
	}
	resource.Params[key] = val
	return
}

func (resource *Resource) AppendPre(pre ResourcePre) {
	resource.pres = append(resource.pres, pre)
	return
}

func (resource *Resource) Pre() {
	if resource.pres == nil {
		return
	}
	pres := resource.pres
	resource.pres = nil
	for _, pre := range pres {
		pre(resource)
	}
	return
}

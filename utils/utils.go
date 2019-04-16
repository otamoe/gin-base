package utils

import (
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
)

func GetContextValue(ctx *gin.Context, keys []string) (value interface{}, ok bool) {
	if len(keys) == 0 {
		return
	}

	// keys[] 不存在
	if value, ok = ctx.Get(keys[0]); !ok || value == nil {
		return
	}

	dataV := reflect.ValueOf(value)

	if dataV.Kind() == reflect.Interface {
		dataV = dataV.Elem()
	}

	for i, name := range keys {
		if i == 0 {
			continue
		}
		ok = false
		if dataV.IsNil() {
			break
		}
		if dataV.Kind() == reflect.Ptr {
			dataV = dataV.Elem()
		}
		if dataV.Kind() != reflect.Struct {
			break
		}
		dataV = dataV.FieldByName(name)
		ok = true
	}

	value = dataV.Interface()
	return
}

func NameOfFunction(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

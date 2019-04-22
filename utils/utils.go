package utils

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	RandNumber = []byte("0123456789")

	RandAlpha       = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	RandAlphaNumber = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	RandAlphaLower       = []byte("abcdefghijklmnopqrstuvwxyz")
	RandAlphaLowerNumber = []byte("0123456789abcdefghijklmnopqrstuvwxyz")
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

func IsEmpty(val interface{}) bool {
	return val == nil || val == 0 || val == "" || val == false
}

func RandInt(n int) int {
	bn, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err)
	}
	return int(bn.Int64())
}

func RandInt64(n int64) int64 {
	bn, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		panic(err)
	}
	return bn.Int64()
}

func RandRune(n int, runes []rune) []rune {
	b := make([]rune, n)
	for i := range b {
		bn, err := rand.Int(rand.Reader, big.NewInt(int64(len(runes))))
		if err != nil {
			panic(err)
		}
		b[i] = runes[bn.Int64()]
	}
	return b
}

func RandByte(n int, bytes []byte) []byte {
	b := make([]byte, n)
	for i := range b {
		bn, err := rand.Int(rand.Reader, big.NewInt(int64(len(bytes))))
		if err != nil {
			panic(err)
		}
		b[i] = bytes[bn.Int64()]
	}
	return b
}

func IsMobile(req *http.Request) bool {
	ua := req.Header.Get("user-agent")
	if ua == "" {
		return false
	}
	if strings.Index(ua, "Mobile") != -1 {
		return true
	}
	if strings.Index(ua, "Android") != -1 {
		return true
	}
	if strings.Index(ua, "Silk/") != -1 {
		return true
	}
	if strings.Index(ua, "Kindle") != -1 {
		return true
	}
	if strings.Index(ua, "BlackBerry") != -1 {
		return true
	}
	if strings.Index(ua, "Opera Mini") != -1 {
		return true
	}
	if strings.Index(ua, "Opera Mobi") != -1 {
		return true
	}
	return false
}

package rate

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/otamoe/gin-server/errors"
	redisMiddleware "github.com/otamoe/gin-server/redis"
)

type (
	Config struct {
		Name   string
		IP     bool
		Keys   [][]string
		Filter func(ctx *gin.Context) bool
		Limit  func(ctx *gin.Context) int64
		Reset  time.Duration
	}
)

var CONTEXT = "GIN.ENGINE.RATE"

var PREFIX = "rate"

func Middleware(rates ...Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var err error
		var limit int64
		var remaining int64
		var reset time.Time
		defer func() {
			if err != nil {
				ctx.Error(err)
				ctx.Abort()
				return
			}
			ctx.Next()
		}()
		redisClient := ctx.MustGet(redisMiddleware.CONTEXT).(*redis.Client)

		for i, rate := range rates {
			var key string

			var rateLimit int64
			var rateRemaining int64
			var rateReset time.Time

			if rateLimit = rate.Limit(ctx); rateLimit <= 0 {
				continue
			}

			// keys 获取
			{
				keys := []string{PREFIX, rate.Name, strconv.Itoa(i)}

				// ip
				if rate.IP {
					keys = append(keys, base64.StdEncoding.EncodeToString([]byte(ctx.ClientIP())))
				}

				// keys
				for _, val := range rate.Keys {
					keys = append(keys, getValue(ctx, val))
				}

				key = strings.Join(keys, ".")
			}

			// 剩余
			var cmd *redis.IntCmd
			_, err = redisClient.Pipelined(func(pipe redis.Pipeliner) error {
				pipe.SetNX(key, 0, rate.Reset)
				cmd = pipe.Incr(key)
				return nil
			})
			if err != nil {
				return
			}
			var usedValue int64
			if usedValue, err = cmd.Result(); err != nil {
				return
			}

			if usedValue < 1 {
				usedValue = 1
			}
			rateRemaining = rateLimit - usedValue

			// ttl
			var ttl time.Duration
			if ttl, err = redisClient.TTL(key).Result(); err != nil {
				return
			}

			if ttl > 0 {
				rateReset = rateReset.Add(ttl)
			} else {
				rateReset = rateReset.Add(rate.Reset)
			}

			if limit == 0 || remaining > rateRemaining {
				remaining = rateRemaining
			}
			if limit == 0 || rateLimit < limit {
				limit = rateLimit
			}
			if reset.Before(rateReset) {
				reset = rateReset
			}

			defer func(rate Config, key string) {
				if rate.Filter == nil {
					return
				}
				if !rate.Filter(ctx) {
					return
				}
				// 滤器掉的 减少
				var cmd *redis.IntCmd
				redisClient.Pipelined(func(pipe redis.Pipeliner) error {
					pipe.SetNX(key, 0, rate.Reset)
					cmd = pipe.Decr(key)
					return nil
				})
				cmd.Result()
			}(rate, key)
		}

		if limit != 0 && reset.Unix() > 0 {
			if remaining < 0 {
				remaining = 0
			}
			ctx.Header("X-RateLimit-Limit", strconv.FormatInt(limit, 10))
			ctx.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
			ctx.Header("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
			if remaining == 0 && gin.Mode() != gin.DebugMode {
				err = &errors.Error{
					Message:    http.StatusText(http.StatusTooManyRequests),
					Type:       "rate",
					StatusCode: http.StatusTooManyRequests,
					Params: map[string]interface{}{
						"limit": limit,
						"reset": reset,
					},
				}
				return
			}
		}
	}
}

func getValue(ctx *gin.Context, keys []string) (value string) {
	if len(keys) == 0 {
		return
	}
	val, ok := ctx.Get(keys[0])
	if !ok || val == nil {
		return
	}

	dataV := reflect.ValueOf(val)

	if dataV.Kind() == reflect.Interface {
		dataV = dataV.Elem()
	}

	for i, name := range keys {
		if i == 0 {
			continue
		}
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
	}

	val = dataV.Interface()

	hash := md5.New()
	switch val.(type) {
	case string:
		value = val.(string)
		hash.Write([]byte(value))
	default:
		hash := md5.New()
		valBytes, _ := json.Marshal(val)
		hash.Write(valBytes)
	}
	value = base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return
}

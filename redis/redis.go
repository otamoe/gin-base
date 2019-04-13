package redis

import (
	"github.com/go-redis/redis"

	"github.com/gin-gonic/gin"
)

type (
	GetSession func() *redis.Client
)

var CONTEXT = "GIN.ENGINE.REDIS"

func Middleware(getSession GetSession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session := getSession()
		defer session.Close()
		ctx.Set(CONTEXT, session)
		ctx.Next()
	}
}

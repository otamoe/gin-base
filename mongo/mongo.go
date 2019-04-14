package mongo

import (
	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
)

type (
	GetSession func() *mgo.Session
)

var CONTEXT = "GIN.SERVER.MONGO"

func Middleware(getSession GetSession) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session := getSession()
		defer session.Close()
		ctx.Set(CONTEXT, session)
		ctx.Next()
	}
}

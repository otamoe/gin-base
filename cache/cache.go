package cache

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type (
	Config struct {
		Control []string
	}
	Cache struct {
		Control      []string
		context      *gin.Context
		Etag         interface{}
		LastModified *time.Time
	}
)

var CONTEXT = "GIN.SERVER.CACHE"

func Middleware(c Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set(CONTEXT, &Cache{
			Control: c.Control,
			context: ctx,
		})
		ctx.Next()
	}
}

func (c *Cache) Header() {
	ctx := c.context
	ctx.Header("cache-control", strings.Join(c.Control, ","))
	if c.LastModified != nil {
		ctx.Header("last-modified", c.LastModified.Format(http.TimeFormat))
	}
	if c.Etag != nil {
		ctx.Header("etag", "\""+fmt.Sprint(c.Etag)+"\"")
	} else if c.LastModified != nil {
		ctx.Header("etag", "\""+fmt.Sprint(c.LastModified.Unix())+"\"")
	}
}

func (c *Cache) Match() bool {
	c.Header()
	ctx := c.context
	ifUnmodifiedSince := ctx.GetHeader("if-unmodified-since")
	ifModifiedSince := ctx.GetHeader("if-modified-since")

	ifMatch := ctx.GetHeader("if-match")
	ifNoneMatch := ctx.GetHeader("if-none-match")

	etag := ctx.Writer.Header().Get("etag")
	lastModified := ctx.Writer.Header().Get("last-modified")

	if ifUnmodifiedSince != "" && ifUnmodifiedSince != lastModified {
		ctx.AbortWithStatus(http.StatusPreconditionFailed)
		return true
	}

	if ifMatch != "" && ifMatch != etag {
		ctx.AbortWithStatus(http.StatusPreconditionFailed)
		return true
	}

	if ifModifiedSince != "" && ifModifiedSince != lastModified {
		return false
	}
	if ifNoneMatch != "" && ifNoneMatch != etag {
		return false
	}
	if ifNoneMatch == "" && ifUnmodifiedSince == "" {
		return false
	}
	ctx.AbortWithStatus(http.StatusNotModified)
	return true
}

package logger

import (
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/otamoe/gin-engine/name"
	mgoModel "github.com/otamoe/mgo-model"
	"github.com/sirupsen/logrus"
)

type (
	Config struct {
		Prefix string
	}
	Logger struct {
		mgoModel.DocumentBase `json:"-" bson:"-" binding:"-"`
		ID                    bson.ObjectId          `json:"_id" bson:"_id"`
		UserID                bson.ObjectId          `json:"user_id,omitempty" bson:"user,omitempty"`
		TokenID               bson.ObjectId          `json:"token_id,omitempty" bson:"token,omitempty"`
		Type                  string                 `json:"type,omitempty" bson:"type,omitempty"`
		Action                string                 `json:"action,omitempty" bson:"action,omitempty"`
		IP                    string                 `json:"ip,omitempty" bson:"ip,omitempty"`
		Method                string                 `json:"method,omitempty" bson:"method,omitempty"`
		Scheme                string                 `json:"scheme,omitempty" bson:"scheme,omitempty"`
		Host                  string                 `json:"host,omitempty" bson:"host,omitempty"`
		Path                  string                 `json:"path,omitempty" bson:"path,omitempty"`
		Query                 url.Values             `json:"query,omitempty" bson:"query,omitempty"`
		Params                map[string]string      `json:"params,omitempty" bson:"params,omitempty"`
		Bind                  interface{}            `json:"bind,omitempty" bson:"bind,omitempty"`
		Latency               time.Duration          `json:"latency,omitempty" bson:"latency,omitempty"`
		StatusCode            int                    `json:"status_code,omitempty" bson:"status_code,omitempty"`
		ErrorsText            string                 `json:"errors_text,omitempty" bson:"errors_text,omitempty"`
		Fields                map[string]interface{} `json:"fields,omitempty" bson:"fields,omitempty"`
		CreatedAt             *time.Time             `json:"created_at" bson:"created_at"`
	}
)

var (
	CONTEXT        = "GIN.ENGINE.LOGGER"
	CONTEXT_FIELDS = "GIN.ENGINE.LOGGER.FIELDS"
	Model          = &mgoModel.Model{
		Name:     "loggers",
		Document: &Logger{},
		Indexs: []mgo.Index{
			mgo.Index{
				Key:        []string{"created_at"},
				Background: true,
			},
		},
	}
)

func Middleware(c Config) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		req := ctx.Request

		now2 := time.Now()
		now := &now2

		url := req.URL

		var host string
		if host = req.Header.Get("X-Forwarded-Host"); host != "" {
		} else if host = req.Header.Get("X-Host"); host != "" {
		} else if host = req.Host; host != "" {
		} else if host = url.Host; host != "" {
		}

		logger := &Logger{
			ID:        bson.NewObjectId(),
			IP:        ctx.ClientIP(),
			Method:    req.Method,
			Scheme:    url.Scheme,
			Host:      host,
			Path:      url.Path,
			Query:     url.Query(),
			CreatedAt: now,
		}

		ctx.Set(CONTEXT, logger)

		defer func() {

			//  被删除
			if val, ok := ctx.Get(CONTEXT); !ok || val == nil || val == false {
				return
			}

			if logger.StatusCode == 0 {
				logger.StatusCode = ctx.Writer.Status()
			}

			if logger.Latency == 0 {
				logger.Latency = time.Now().Sub(*now)
			}

			if logger.Type == "" {
				if val, ok := ctx.Get(name.CONTEXT_TYPE); ok {
					if typ, ok := val.(string); ok {
						logger.Type = typ
					}
				}
			}

			if logger.Action == "" {
				if val, ok := ctx.Get(name.CONTEXT_ACTION); ok {
					if action, ok := val.(string); ok {
						logger.Action = action
					}
				}
			}

			if logger.Params == nil && len(ctx.Params) != 0 {
				logger.Params = map[string]string{}
				for _, param := range ctx.Params {
					logger.Params[param.Key] = param.Value
				}
			}

			// OPTIONS 请求 忽略
			if logger.Method == "OPTIONS" && logger.StatusCode < 500 {
				return
			}

			// 错误消息
			logger.ErrorsText = strings.TrimSpace(ctx.Errors.ByType(gin.ErrorTypeAny).String())

			// 错误信息加上 请求头
			if logger.StatusCode >= 500 {
				httprequest, _ := httputil.DumpRequest(ctx.Request, false)
				logger.ErrorsText += "\n" + strings.TrimSpace(string(httprequest))
			}

			if val, ok := ctx.Get(CONTEXT_FIELDS); ok {
				if val, ok := val.(map[string]interface{}); ok {
					if logger.Fields == nil {
						logger.Fields = val
					} else {
						for k, v := range val {
							logger.Fields[k] = v
						}
					}
				}
			}

			if logger.Fields == nil {
				logger.Fields = map[string]interface{}{}
			}

			logger.Fields["_ip"] = logger.IP
			logger.Fields["_latency"] = logger.Latency
			logger.Fields["_token"] = logger.TokenID
			logger.Fields["_user"] = logger.UserID
			logger.Fields["_bind"] = logger.Bind
			logger.Fields["_params"] = logger.Params

			rawPath := logger.Path
			if val := logger.Query.Encode(); val != "" {
				rawPath += "?" + val
			}

			with := logrus.WithFields(logger.Fields)

			if logger.StatusCode >= 500 {
				with.Errorf("%s%s %s %d %s\n%s\n", c.Prefix, logger.ID, logger.Method, logger.StatusCode, rawPath, logger.ErrorsText)
			} else if logger.ErrorsText != "" {
				with.Warnf("%s%s %s %d %s\n%s\n", c.Prefix, logger.ID, logger.Method, logger.StatusCode, rawPath, logger.ErrorsText)
			} else {
				with.Infof("%s%s %s %d %s", c.Prefix, logger.ID, logger.Method, logger.StatusCode, rawPath)
			}
		}()
		ctx.Next()
	}
}

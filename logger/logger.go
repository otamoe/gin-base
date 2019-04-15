package logger

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/otamoe/gin-server/bind"
	"github.com/otamoe/gin-server/resource"
	mgoModel "github.com/otamoe/mgo-model"
	"github.com/sirupsen/logrus"
)

type (
	Config struct {
		Prefix string
		Logger *logrus.Logger
	}
	Logger struct {
		mgoModel.DocumentBase `json:"-" bson:"-" binding:"-"`
		ID                    bson.ObjectId          `json:"_id" bson:"_id"`
		UserID                bson.ObjectId          `json:"user_id,omitempty" bson:"user,omitempty"`
		TokenID               bson.ObjectId          `json:"token_id,omitempty" bson:"token,omitempty"`
		Handler               string                 `json:"handler,omitempty" bson:"handler,omitempty"`
		Type                  string                 `json:"type,omitempty" bson:"type,omitempty"`
		Action                string                 `json:"action,omitempty" bson:"action,omitempty"`
		Value                 string                 `json:"value,omitempty" bson:"value,omitempty"`
		IP                    string                 `json:"ip,omitempty" bson:"ip,omitempty"`
		Method                string                 `json:"method,omitempty" bson:"method,omitempty"`
		Scheme                string                 `json:"scheme,omitempty" bson:"scheme,omitempty"`
		Host                  string                 `json:"host,omitempty" bson:"host,omitempty"`
		Path                  string                 `json:"path,omitempty" bson:"path,omitempty"`
		Query                 url.Values             `json:"query,omitempty" bson:"query,omitempty"`
		Params                map[string]string      `json:"params,omitempty" bson:"params,omitempty"`
		Bind                  map[string]interface{} `json:"bind,omitempty" bson:"bind,omitempty"`
		Latency               time.Duration          `json:"latency,omitempty" bson:"latency,omitempty"`
		StatusCode            int                    `json:"status_code,omitempty" bson:"status_code,omitempty"`
		ErrorsText            string                 `json:"errors_text,omitempty" bson:"errors_text,omitempty"`
		Fields                map[string]interface{} `json:"fields,omitempty" bson:"fields,omitempty"`
		CreatedAt             *time.Time             `json:"created_at" bson:"created_at"`
		Logrus                *logrus.Logger         `json:"-" bson:"-" binding:"-"`
	}
	BindInterface interface {
		BindMarshal() map[string]interface{}
	}
)

var (
	CONTEXT          = "GIN.SERVER.LOGGER"
	CONTEXT_CALLBACK = "GIN.SERVER.LOGGER.CALBACK"
	Model            = &mgoModel.Model{
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
			Fields:    map[string]interface{}{},
			CreatedAt: now,
			Logrus:    c.Logger,
		}

		ctx.Set(CONTEXT, logger)

		defer func() {

			//  被删除
			if val, ok := ctx.Get(CONTEXT); !ok || val == nil || val == false {
				return
			}

			// OPTIONS 请求 忽略
			if logger.Method == "OPTIONS" && ctx.Writer.Status() < http.StatusInternalServerError {
				return
			}

			if logger.StatusCode == 0 {
				logger.StatusCode = ctx.Writer.Status()
			}

			if logger.Latency == 0 {
				logger.Latency = time.Now().Sub(*now)
			}

			if logger.Handler == "" {
				logger.Handler = ctx.GetString(resource.CONTEXT_HANDLER)
			}

			if logger.Type == "" {
				logger.Type = ctx.GetString(resource.CONTEXT_TYPE)
			}

			if logger.Action == "" {
				logger.Action = ctx.GetString(resource.CONTEXT_ACTION)
			}

			if logger.Params == nil && len(ctx.Params) != 0 {
				logger.Params = map[string]string{}
				for _, param := range ctx.Params {
					logger.Params[param.Key] = param.Value
				}
			}

			// bind
			if logger.Bind == nil {
				if val, ok := ctx.Get(bind.CONTEXT); ok && val != nil {
					if bindInterface, ok := val.(BindInterface); ok {
						logger.Bind = bindInterface.BindMarshal()
					} else if jsonBytes, err := json.Marshal(val); err == nil {
						logger.Bind = map[string]interface{}{}
						json.Unmarshal(jsonBytes, logger.Bind)
					}
				}
			}

			// 错误消息
			logger.ErrorsText = strings.TrimSpace(ctx.Errors.ByType(gin.ErrorTypeAny).String())

			// 错误信息加上 请求头
			if logger.StatusCode >= 500 {
				httprequest, _ := httputil.DumpRequest(ctx.Request, false)
				logger.ErrorsText += "\n" + strings.TrimSpace(string(httprequest))
			}

			logger.Fields["ip"] = logger.IP
			logger.Fields["latency"] = logger.Latency

			if logger.TokenID != "" {
				logger.Fields["token_id"] = logger.TokenID.Hex()
			}

			if logger.UserID != "" {
				logger.Fields["user_id"] = logger.UserID.Hex()
			}

			if len(logger.Params) != 0 {
				for name, val := range logger.Params {
					logger.Fields["param_"+name] = val
				}
			}

			if logger.Bind != nil {
				for name, val := range logger.Bind {
					logger.Fields["bind_"+name] = val
				}
			}

			if logger.ErrorsText != "" {
				logger.Fields["errors_text"] = logger.ErrorsText
			}

			rawPath := logger.Path
			if val := logger.Query.Encode(); val != "" {
				rawPath += "?" + val
			}

			with := logger.Logrus.WithFields(logger.Fields)
			// callback
			if val, ok := ctx.Get(CONTEXT_CALLBACK); ok && val != nil {
				if call, ok := val.(func(*Logger)); ok {
					call(logger)
				}
			}

			if logger.StatusCode >= 500 {
				with.Errorf("%s%s %s/%s/%s/%s %s %d %s", c.Prefix, logger.ID.Hex(), logger.Handler, logger.Type, logger.Action, logger.Value, logger.Method, logger.StatusCode, rawPath)
			} else if logger.ErrorsText != "" {
				with.Warnf("%s%s %s/%s/%s/%s %s %d %s", c.Prefix, logger.ID.Hex(), logger.Handler, logger.Type, logger.Action, logger.Value, logger.Method, logger.StatusCode, rawPath)
			} else {
				with.Infof("%s%s %s/%s/%s/%s %s %d %s", c.Prefix, logger.ID.Hex(), logger.Handler, logger.Type, logger.Action, logger.Value, logger.Method, logger.StatusCode, rawPath)
			}
		}()
		ctx.Next()
	}
}

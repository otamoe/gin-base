package server

import (
	"log"
	"strings"
	"sync"
	"time"

	"github.com/globalsign/mgo"
	"github.com/otamoe/gin-server/mongo"
	mgoModel "github.com/otamoe/mgo-model"
)

type (
	Mongo struct {
		URLs []string `json:"urls,omitempty"`

		PoolLimit   int           `json:"pool_limit,omitempty"`
		PoolTimeout time.Duration `json:"pool_timeout,omitempty"`

		DialTimeout   time.Duration `json:"dial_timeout,omitempty"`
		SocketTimeout time.Duration `json:"socket_timeout,omitempty"`

		session *mgo.Session
		once    sync.Once
	}
)

func init() {
	mgoModel.CONTEXT = mongo.CONTEXT
}

func (config *Mongo) init(server *Server, handler *Handler) {
	if config.session != nil {
		return
	}
	if len(config.URLs) == 0 {
		var db string
		if handler != nil && handler.Name != "" {
			db = handler.Name
		} else if server != nil && server.Name != "" {
			db = server.Name
		}
		config.URLs = append(config.URLs, "localhost:27017/"+db)
	}
	if config.PoolLimit == 0 {
		config.PoolLimit = 2048
	}
	if config.PoolTimeout == 0 {
		config.PoolTimeout = time.Second * 3
	}
	if config.DialTimeout == 0 {
		config.DialTimeout = time.Second * 2
	}
	if config.SocketTimeout == 0 {
		config.SocketTimeout = time.Minute * 1
	}

	if handler == nil && server != nil {
		if server.ENV == "development" {
			mgo.SetDebug(true)
			logWriter := server.Logger.Get().Writer()
			mgo.SetLogger(log.New(logWriter, "", 0))
		}
	}

	var err error
	if config.session, err = mgo.DialWithTimeout(strings.Join(config.URLs, ","), config.DialTimeout); err != nil {
		panic(err)
	}
	config.session.SetPoolLimit(config.PoolLimit)
	config.session.SetPoolTimeout(config.PoolTimeout)
	config.session.SetSocketTimeout(config.SocketTimeout)
}

func (config *Mongo) Get() *mgo.Session {
	return config.session.Clone()
}

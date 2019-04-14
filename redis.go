package server

import (
	"log"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

type (
	Redis struct {
		URLs []string `json:"urls,omitempty"`

		PoolLimit   int           `json:"pool_limit,omitempty"`
		PoolTimeout time.Duration `json:"pool_timeout,omitempty"`

		DialTimeout   time.Duration `json:"dial_timeout,omitempty"`
		SocketTimeout time.Duration `json:"socket_timeout,omitempty"`
	}
)

func (config *Redis) init(server *Server, handler *Handler) {
	if len(config.URLs) == 0 {
		config.URLs = append(config.URLs, "localhost:6379")
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
		config.SocketTimeout = time.Second * 2
	}
	if handler == nil && server != nil {
		logWriter := server.Logger.Get().Writer()
		redis.SetLogger(log.New(logWriter, "", 0))
	}
}

func (config *Redis) Get() (client *redis.Client) {
	client = redis.NewClient(&redis.Options{
		Addr:         strings.Join(config.URLs, ","),
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.SocketTimeout,
		WriteTimeout: config.SocketTimeout,
		PoolSize:     config.PoolLimit,
		PoolTimeout:  config.PoolTimeout,
	})
	return
}

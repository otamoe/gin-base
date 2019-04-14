package server

import (
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

type (
	Logger struct {
		File   string `json:"file,omitempty"`
		logger *logrus.Logger
	}
)

func (config *Logger) init(server *Server, handler *Handler) {
	if config.logger != nil {
		return
	}
	if handler == nil {
		config.logger = logrus.StandardLogger()
	} else {
		config.logger = logrus.New()
	}
	if server != nil {
		switch server.ENV {
		case "development":
			config.logger.SetLevel(logrus.TraceLevel)
		case "test":
			config.logger.SetLevel(logrus.TraceLevel)
		default:
			config.logger.SetLevel(logrus.InfoLevel)
		}
	}

	config.logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})

	config.logger.SetOutput(os.Stdout)
	if config.File != "" {
		writer, err := os.OpenFile(config.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		config.logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
		config.logger.SetOutput(writer)
	}

	if handler == nil {
		log.SetOutput(logrus.StandardLogger().Writer())
	}
}

func (config *Logger) Get() *logrus.Logger {
	return config.logger
}

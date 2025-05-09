package mongo

import (
	"fmt"

	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	Host   string
	Port   string
	DBName string
}

func (c *Config) conString() string {
	return fmt.Sprintf("mongodb://%s:%s/", c.Host, c.Port)
}

func (c *Config) Options() *options.ClientOptions {
	return options.Client().ApplyURI(c.conString())
}

package config

import (
	"os"

	"github.com/anyproto/any-sync/accountservice"
	"github.com/anyproto/any-sync/app"
	"github.com/anyproto/any-sync/net/rpc"
	"github.com/anyproto/any-sync/net/secureservice"
	"github.com/anyproto/any-sync/net/transport/quic"
	"github.com/anyproto/any-sync/net/transport/yamux"
	"github.com/anyproto/any-sync/nodeconf"
	"gopkg.in/yaml.v3"

	"github.com/anyproto/anytype-push-server/db"
	"github.com/anyproto/anytype-push-server/redisprovider"
	"github.com/anyproto/anytype-push-server/sender/provider/fcm"
)

const CName = "config"

func NewFromFile(path string) (c *Config, err error) {
	c = &Config{}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}
	return
}

type Config struct {
	Account                  accountservice.Config  `yaml:"account"`
	Mongo                    db.Mongo               `yaml:"mongo"`
	Redis                    redisprovider.Config   `yaml:"redis"`
	Drpc                     rpc.Config             `yaml:"drpc"`
	Yamux                    yamux.Config           `yaml:"yamux"`
	Quic                     quic.Config            `yaml:"quic"`
	Network                  nodeconf.Configuration `yaml:"network"`
	NetworkStorePath         string                 `yaml:"networkStorePath"`
	NetworkUpdateIntervalSec int                    `yaml:"networkUpdateIntervalSec"`
	FCM                      fcm.Config             `yaml:"fcm"`
}

func (c *Config) Init(a *app.App) (err error) {
	return nil
}

func (c *Config) Name() (name string) {
	return CName
}

func (c *Config) GetMongo() db.Mongo {
	return c.Mongo
}

func (c *Config) GetDrpc() rpc.Config {
	return c.Drpc
}

func (c *Config) GetAccount() accountservice.Config {
	return c.Account
}

func (c *Config) GetNodeConf() nodeconf.Configuration {
	return c.Network
}

func (c *Config) GetNodeConfStorePath() string {
	return c.NetworkStorePath
}

func (c *Config) GetNodeConfUpdateInterval() int {
	return c.NetworkUpdateIntervalSec
}

func (c *Config) GetYamux() yamux.Config {
	return c.Yamux
}

func (c *Config) GetQuic() quic.Config {
	return c.Quic
}

func (c *Config) GetSecureService() secureservice.Config {
	return secureservice.Config{RequireClientAuth: true}
}

func (c *Config) GetRedis() redisprovider.Config {
	return c.Redis
}

func (c *Config) GetFCM() fcm.Config {
	return c.FCM
}

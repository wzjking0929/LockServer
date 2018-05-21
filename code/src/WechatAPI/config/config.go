package config

import (
	"flag"
	"io/ioutil"
	"os"
	"sync"

	log "github.com/Sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

var configOpt Option
var onceDataOpt sync.Once

func InitConfig() {
	onceDataOpt.Do(func() {
		configOpt = loadConfig()
	})
}

type Option struct {
	Database string `yaml:"Database"`

	LogFile    string `yaml:"LogFile"`
	LogLevel   string `yaml:"LogLevel"`
	SysLogAddr string `yaml:"SysLogAddr"`

	RedisAddr    string `yaml:"RedisAddr"`
	RedisPwd     string `yaml:"RedisPwd"`
	RedisTimeOut int    `yaml:"RedisTimeOut"`

	RecvAmqpURI       string `yaml:"AmqpURI"`
	RecvExchangeName  string `yaml:"RecvExchangeName"`
	RecvChanReadQName string `yaml:"RecvChanReadQName"`
	RecvRoutKey       string `yaml:"RecvRoutKey"`

	PushblishAmqpURI string `yaml:"PushblishAmqpURI"`
	PushblisRoutKey  string `yaml:"PushblisRoutKey"`
}

func loadConfig() (p Option) {
	var confName string
	flag.StringVar(&confName, "f", "config.yml", "config file")
	flag.Parse()
	f, err := os.Open(confName)
	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal("read config file err: " + err.Error())
	}
	err = yaml.Unmarshal(data, &p)
	if err != nil {
		log.Fatal("unmarshal yaml config err: " + err.Error())
	}
	return p
}

//GetConfig get global config
func GetConfig() *Option {
	return &configOpt
}

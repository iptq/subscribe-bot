package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Debug        bool   `toml:"debug"`
	BotToken     string `toml:"bot_token"`
	ClientId     int    `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
	Repos        string `toml:"repos"`
	DatabasePath string `toml:"db_path"`

	Web WebConfig `toml:"web"`
}

type WebConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

func ReadConfig(path string) (config Config, err error) {
	file, err := os.Open(path)
	if err != nil {
		err = fmt.Errorf("couldn't open file %s: %w", path, err)
		return
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		err = fmt.Errorf("couldn't read data from %s: %w", path, err)
		return
	}

	err = toml.Unmarshal(data, &config)
	if err != nil {
		err = fmt.Errorf("couldn't parse config data from %s: %w", path, err)
		return
	}

	return
}
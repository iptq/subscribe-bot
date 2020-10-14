package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Debug        bool   `toml:"debug,omitempty"`
	BotToken     string `toml:"bot_token"`
	Repos        string `toml:"repos"`
	DatabasePath string `toml:"db_path"`

	Oauth OauthConfig `toml:"oauth"`
	Web   WebConfig   `toml:"web"`
}

type OauthConfig struct {
	ClientId     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
}

type WebConfig struct {
	Host          string `toml:"host"`
	Port          int    `toml:"port"`
	ServedAt      string `toml:"served_at"`
	SessionSecret string `toml:"session_secret"`
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

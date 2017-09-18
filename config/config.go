package config

import (
	"io/ioutil"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	toml "github.com/pelletier/go-toml"
)

type Config struct {
	SlackURL string
	Channel  string
	Username string
}

func NewConfig() *Config {
	return &Config{
		SlackURL: "",
		Channel:  "",
		Username: "",
	}
}

func (c *Config) LoadTOML(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	config, err := toml.LoadBytes(b)
	if err != nil {
		return err
	}

	slackConfig := config.Get("slack").(*toml.Tree)

	if c.SlackURL == "" {
		c.SlackURL = slackConfig.Get("url").(string)
	}
	if c.Channel == "" {
		c.Channel = slackConfig.Get("channel").(string)
	}
	if c.Username == "" {
		c.Username = slackConfig.Get("username").(string)
	}

	return nil
}

func LoadTOMLFilename(filename string) string {
	if filename != "" {
		return filename
	}

	homeDir, err := homedir.Dir()
	if err == nil {
		tomlFile := homeDir + "/etc/notify_slack.toml"
		if fileExists(tomlFile) {
			return tomlFile
		}
	}

	tomlFile := "/etc/notify_slack.toml"
	if fileExists(tomlFile) {
		return tomlFile
	}

	return ""
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)

	return err == nil
}

package config

import (
	"io/ioutil"
	"os"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	toml "github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

type Config struct {
	SlackURL  string
	Token     string
	Channel   string
	Username  string
	IconEmoji string
	Duration  time.Duration
}

func NewConfig() *Config {
	return &Config{
		SlackURL:  "",
		Token:     "",
		Channel:   "",
		Username:  "",
		IconEmoji: "",
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
		slackURL, ok := slackConfig.Get("url").(string)
		if ok {
			c.SlackURL = slackURL
		}
	}
	if c.Token == "" {
		token, ok := slackConfig.Get("token").(string)
		if ok {
			c.Token = token
		}
	}
	if c.Channel == "" {
		channel, ok := slackConfig.Get("channel").(string)
		if ok {
			c.Channel = channel
		}
	}
	if c.Username == "" {
		username, ok := slackConfig.Get("username").(string)
		if ok {
			c.Username = username
		}
	}
	if c.IconEmoji == "" {
		iconEmoji, ok := slackConfig.Get("icon_emoji").(string)
		if ok {
			c.IconEmoji = iconEmoji
		}
	}

	durationStr, ok := slackConfig.Get("interval").(string)
	if ok {
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return errors.Wrapf(err, "incorrect value to inteval option: %s", durationStr)
		}
		c.Duration = duration
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

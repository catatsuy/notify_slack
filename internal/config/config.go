package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	toml "github.com/pelletier/go-toml/v2"
)

var (
	userHomeDir = os.UserHomeDir
)

type Config struct {
	SlackURL       string
	Token          string
	PrimaryChannel string
	Channel        string
	SnippetChannel string
	FileChannelID  string
	Username       string
	IconEmoji      string
	Duration       time.Duration
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) LoadEnv() error {
	if c.SlackURL == "" {
		c.SlackURL = os.Getenv("NOTIFY_SLACK_WEBHOOK_URL")
	}

	if c.Token == "" {
		c.Token = os.Getenv("NOTIFY_SLACK_TOKEN")
	}

	if c.Channel == "" {
		c.Channel = os.Getenv("NOTIFY_SLACK_CHANNEL")
	}

	if c.SnippetChannel == "" {
		if os.Getenv("NOTIFY_SLACK_SNIPPET_CHANNEL") != "" {
			return fmt.Errorf("the NOTIFY_SLACK_SNIPPET_CHANNEL option is deprecated")
		}
	}

	if c.FileChannelID == "" {
		c.FileChannelID = os.Getenv("NOTIFY_SLACK_FILE_CHANNEL_ID")
	}

	if c.Username == "" {
		c.Username = os.Getenv("NOTIFY_SLACK_USERNAME")
	}

	if c.IconEmoji == "" {
		c.IconEmoji = os.Getenv("NOTIFY_SLACK_ICON_EMOJI")
	}

	durationStr := os.Getenv("NOTIFY_SLACK_INTERVAL")
	if durationStr != "" {
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return fmt.Errorf("incorrect value to inteval option from NOTIFY_SLACK_INTERVAL: %s: %w", durationStr, err)
		}
		c.Duration = duration
	}

	return nil
}

type slackConfig struct {
	URL            string
	Token          string
	Channel        string
	SnippetChannel string `toml:"snippet_channel"`
	FileChannelID  string `toml:"file_channel_id"`
	Username       string
	IconEmoji      string `toml:"icon_emoji"`
	Interval       string
}

type rootConfig struct {
	Slack slackConfig
}

func (c *Config) LoadTOML(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	var cfg rootConfig

	err = toml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return err
	}

	slackConfig := cfg.Slack

	if c.SlackURL == "" {
		if slackConfig.URL != "" {
			c.SlackURL = slackConfig.URL
		}
	}
	if c.Token == "" {
		if slackConfig.Token != "" {
			c.Token = slackConfig.Token
		}
	}
	if c.Channel == "" {
		if slackConfig.Channel != "" {
			c.Channel = slackConfig.Channel
		}
	}
	if c.Username == "" {
		if slackConfig.Username != "" {
			c.Username = slackConfig.Username
		}
	}
	if slackConfig.SnippetChannel != "" {
		return fmt.Errorf("the snippet_channel option is deprecated")
	}
	if c.FileChannelID == "" {
		if slackConfig.FileChannelID != "" {
			c.FileChannelID = slackConfig.FileChannelID
		}
	}
	if c.IconEmoji == "" {
		if slackConfig.IconEmoji != "" {
			c.IconEmoji = slackConfig.IconEmoji
		}
	}

	if slackConfig.Interval != "" {
		duration, err := time.ParseDuration(slackConfig.Interval)
		if err != nil {
			return fmt.Errorf("incorrect value to interval option: %s: %w", slackConfig.Interval, err)
		}
		c.Duration = duration
	}

	return nil
}

func LoadTOMLFilename(filename string) string {
	if filename != "" {
		return filename
	}

	homeDir, err := userHomeDir()
	if err == nil {
		tomlFile := filepath.Join(homeDir, ".notify_slack.toml")
		if fileExists(tomlFile) {
			return tomlFile
		}

		tomlFile = filepath.Join(homeDir, "/etc/notify_slack.toml")
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

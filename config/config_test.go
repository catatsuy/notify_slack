package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/catatsuy/notify_slack/config"
)

func TestLoadTOML(t *testing.T) {
	c := NewConfig()
	err := c.LoadTOML("./testdata/config.toml")
	if err != nil {
		t.Fatal(err)
	}
	expectedSlackURL := "https://hooks.slack.com/aaaaa"
	if c.SlackURL != expectedSlackURL {
		t.Errorf("got %s, want %s", c.SlackURL, expectedSlackURL)
	}
	expectedToken := "xoxp-token"
	if c.Token != expectedToken {
		t.Errorf("got %s, want %s", c.Token, expectedToken)
	}
	expectedChannel := "#test"
	if c.Channel != expectedChannel {
		t.Errorf("got %s, want %s", c.Channel, expectedChannel)
	}
	expectedSnippetChannel := "#general"
	if c.SnippetChannel != expectedSnippetChannel {
		t.Errorf("got %s, want %s", c.SnippetChannel, expectedSnippetChannel)
	}
	expectedUsername := "deploy!"
	if c.Username != expectedUsername {
		t.Errorf("got %s, want %s", c.Username, expectedUsername)
	}
	expectedIconEmoji := ":rocket:"
	if c.IconEmoji != expectedIconEmoji {
		t.Errorf("got %s, want %s", c.IconEmoji, expectedIconEmoji)
	}
	expectedInterval := time.Duration(2 * time.Second)
	if c.Duration != expectedInterval {
		t.Errorf("got %+v, want %+v", c.Duration, expectedInterval)
	}
}

func TestLoadEnv(t *testing.T) {
	expectedSlackURL := "https://hooks.slack.com/aaaaa"
	expectedToken := "xoxp-token"
	expectedChannel := "#test"
	expectedSnippetChannel := "#general"
	expectedUsername := "deploy!"
	expectedIconEmoji := ":rocket:"
	expectedIntervalStr := "2s"
	expectedInterval := time.Duration(2 * time.Second)

	t.Setenv("NOTIFY_SLACK_WEBHOOK_URL", expectedSlackURL)
	t.Setenv("NOTIFY_SLACK_TOKEN", expectedToken)
	t.Setenv("NOTIFY_SLACK_CHANNEL", expectedChannel)
	t.Setenv("NOTIFY_SLACK_SNIPPET_CHANNEL", expectedSnippetChannel)
	t.Setenv("NOTIFY_SLACK_USERNAME", expectedUsername)
	t.Setenv("NOTIFY_SLACK_ICON_EMOJI", expectedIconEmoji)
	t.Setenv("NOTIFY_SLACK_INTERVAL", expectedIntervalStr)

	c := NewConfig()
	err := c.LoadEnv()
	if err != nil {
		t.Fatal(err)
	}

	if c.SlackURL != expectedSlackURL {
		t.Errorf("got %s, want %s", c.SlackURL, expectedSlackURL)
	}

	if c.Token != expectedToken {
		t.Errorf("got %s, want %s", c.Token, expectedToken)
	}

	if c.Channel != expectedChannel {
		t.Errorf("got %s, want %s", c.Channel, expectedChannel)
	}

	if c.SnippetChannel != expectedSnippetChannel {
		t.Errorf("got %s, want %s", c.SnippetChannel, expectedSnippetChannel)
	}

	if c.Username != expectedUsername {
		t.Errorf("got %s, want %s", c.Username, expectedUsername)
	}

	if c.IconEmoji != expectedIconEmoji {
		t.Errorf("got %s, want %s", c.IconEmoji, expectedIconEmoji)
	}

	if c.Duration != expectedInterval {
		t.Errorf("got %+v, want %+v", c.Duration, expectedInterval)
	}
}

func TestLoadTOMLFilename(t *testing.T) {
	baseDir := "./testdata/"
	defer SetUserHomeDir(baseDir)()

	filename := "etc/notify_slack.toml"
	input := filepath.Join(baseDir, filename)
	_, err := os.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(input)

	tomlFile := LoadTOMLFilename("")
	if !equalFilepath(tomlFile, input) {
		t.Errorf("got %s, want %s", tomlFile, input)
	}

	filename = ".notify_slack.toml"
	input = filepath.Join(baseDir, filename)
	_, err = os.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(input)

	tomlFile = LoadTOMLFilename("")
	if !equalFilepath(tomlFile, input) {
		t.Errorf("got %s, want %s", tomlFile, input)
	}
}

func equalFilepath(input1, input2 string) bool {
	path1, _ := filepath.Abs(input1)
	path2, _ := filepath.Abs(input2)

	return path1 == path2
}

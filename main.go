package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/catatsuy/notify_slack/slack"
)

func main() {
	var (
		channel  string
		username string
		slackURL string
		text     string
	)

	flag.StringVar(&channel, "channel", "", "specify channel")
	flag.StringVar(&slackURL, "slack-url", "", "slack url")
	flag.StringVar(&username, "username", "", "specify username")
	flag.StringVar(&text, "text", "", "text")
	flag.Parse()

	if slackURL == "" {
		return
	}

	if text == "" {
		// 標準入力を待つ
		b, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		text = string(b)
	}

	c, err := slack.NewClient(slackURL, nil)
	if err != nil {
		panic(err)
	}

	param := &slack.SlackPostTextParam{
		Channel:   channel,
		Username:  username,
		Text:      text,
		IconEmoji: ":rocket:",
	}

	err = c.PostText(context.Background(), param)
	if err != nil {
		panic(err)
	}
}

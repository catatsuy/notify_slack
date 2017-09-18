package main

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/catatsuy/notify_slack/slack"
	"github.com/catatsuy/notify_slack/throttle"
	toml "github.com/pelletier/go-toml"
)

func main() {
	var (
		channel  string
		username string
		slackURL string
		tomlFile string
		duration time.Duration
	)

	flag.StringVar(&channel, "channel", "", "specify channel")
	flag.StringVar(&slackURL, "slack-url", "", "slack url")
	flag.StringVar(&username, "username", "", "specify username")
	flag.DurationVar(&duration, "interval", time.Second, "interval")
	flag.StringVar(&tomlFile, "f", "", "config file name")

	flag.Parse()

	if tomlFile != "" {
		b, err := ioutil.ReadFile(tomlFile)
		if err != nil {
			panic(err)
		}
		config, err := toml.LoadBytes(b)
		slackConfig := config.Get("slack").(*toml.Tree)

		if slackURL == "" {
			slackURL = slackConfig.Get("url").(string)
		}
		if channel == "" {
			channel = slackConfig.Get("channel").(string)
		}
		if username == "" {
			username = slackConfig.Get("username").(string)
		}
	}

	if slackURL == "" {
		return
	}

	sClient, err := slack.NewClient(slackURL, nil)
	if err != nil {
		panic(err)
	}

	copyStdin := io.TeeReader(os.Stdin, os.Stdout)

	ex := throttle.NewExec(copyStdin)

	c := make(chan os.Signal, 0)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	param := &slack.PostTextParam{
		Channel:   channel,
		Username:  username,
		IconEmoji: ":rocket:",
	}

	flushCallback := func(_ context.Context, output string) error {
		param.Text = output
		return sClient.PostText(context.Background(), param)
	}

	done := make(chan struct{}, 0)

	doneCallback := func(ctx context.Context, output string) error {
		defer func() {
			done <- struct{}{}
		}()

		return flushCallback(ctx, output)
	}

	interval := time.Tick(duration)
	ctx, cancel := context.WithCancel(context.Background())

	ex.Start(ctx, interval, flushCallback, doneCallback)

	select {
	case <-c:
	case <-ex.Wait():
	}
	cancel()

	<-done
}

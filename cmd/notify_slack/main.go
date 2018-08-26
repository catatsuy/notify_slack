package main

import (
	"context"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/catatsuy/notify_slack/config"
	"github.com/catatsuy/notify_slack/slack"
	"github.com/catatsuy/notify_slack/throttle"
)

func main() {
	var (
		filename string
		tomlFile string
		duration time.Duration
	)

	conf := config.NewConfig()

	flag.StringVar(&conf.Channel, "channel", "", "specify channel")
	flag.StringVar(&conf.SlackURL, "slack-url", "", "slack url")
	flag.StringVar(&conf.Token, "token", "", "token")
	flag.StringVar(&conf.Username, "username", "", "specify username")
	flag.StringVar(&conf.IconEmoji, "icon-emoji", "", "specify icon emoji")

	flag.StringVar(&filename, "upload", "", "upload file")
	flag.DurationVar(&duration, "interval", time.Second, "interval")
	flag.StringVar(&tomlFile, "c", "", "config file name")

	flag.Parse()

	tomlFile = config.LoadTOMLFilename(tomlFile)

	if tomlFile != "" {
		conf.LoadTOML(tomlFile)
	}

	if conf.SlackURL == "" {
		log.Fatal("provide Slack URL")
	}

	sClient, err := slack.NewClient(conf.SlackURL, conf.Token, nil)
	if err != nil {
		log.Fatal(err)
	}

	if filename != "" {
		_, err = os.Stat(filename)
		if err != nil {
			log.Fatalf("%s does not exist", filename)
		}

		content, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatal(err)
		}

		param := &slack.PostFileParam{
			Channel:  conf.Channel,
			Filename: filename,
			Content:  string(content),
		}
		err = sClient.PostFile(context.Background(), param)
		if err != nil {
			log.Fatal(err)
		}

		return
	}

	copyStdin := io.TeeReader(os.Stdin, os.Stdout)

	ex := throttle.NewExec(copyStdin)

	c := make(chan os.Signal, 0)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	param := &slack.PostTextParam{
		Channel:   conf.Channel,
		Username:  conf.Username,
		IconEmoji: conf.IconEmoji,
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

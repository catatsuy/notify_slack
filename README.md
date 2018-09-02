# notify_slack

Notify slack from the command line. It receives standard input and notifies Slack all at once every second (can be changed with the `-interval` option).

## Installation

```
go get github.com/catatsuy/notify_slack/cmd/notify_slack
```

Or you download from [Releases](https://github.com/catatsuy/notify_slack/releases).

If you want to develop, please use the `make`.

```
make bundle
make
```

## usage

`./bin/notify_slack` posts to Slack. You specify the setting in command line option or toml setting file.
If both settings are specified, command line option will always take precedence.

```sh
./bin/output | ./bin/notify_slack
```

`./bin/output` is used for testing. While buffering, to post to slack.

``` sh
./bin/notify_slack README.md
```

You post the file as a snippet. A token is required to use the Slack Web API.


### CLI options

```
-c string
      config file name
-channel string
      specify channel
-icon-emoji string
      specify icon emoji
-interval duration
      interval (default 1s)
-slack-url string
      slack url
-token string
      token
-username string
      specify username
-version
      Print version information and quit
```

### toml configuration file

By default check the following files.

1. a file specified with `-c`
2. `$HOME/etc/notify_slack.toml`
3. `/etc/notify_slack.toml`

The contents of the toml file are as follows. `url` is a required parameter. `token` is necessary if you want to post to snippet.

```toml:notify_slack.toml
[slack]
url = "https://hooks.slack.com/services/**"
token = "xxxxx"
channel = "#general"
username = "tester"
icon_emoji = ":rocket:"
```

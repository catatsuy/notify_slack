# notify_slack

Notify slack from the command line. It receives standard input and notifies Slack all at once every second (can be changed with the `-interval` option).

## usage

```sh
./bin/output | ./bin/notify_slack
```

`./bin/output` is used for testing.

`./bin/notify_slack` posts to Slack. You specify the setting in command line option or toml setting file.
If both settings are specified, command line option will always take precedence.

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
-username string
      specify username
```

### toml configuration file

By default check the following files.

1. a file specified with `-c`
2. `$HOME/etc/notify_slack.toml`
3. `/etc/notify_slack.toml`

The contents of the toml file are as follows.

```toml:notify_slack.toml
[slack]
url = "https://hooks.slack.com/services/**"
channel = "#general"
username = "tester"
icon_emoji = ":rocket:"
```

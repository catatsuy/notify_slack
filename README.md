# notify_slack

Notify slack from the command line. It receives standard input and notifies Slack all at once every second (can be changed with the `-interval` option).

Please watch this video. https://www.youtube.com/watch?v=wmKSr9Aoz-Y

## Installation

```
GO111MODULE=on go get github.com/catatsuy/notify_slack/cmd/notify_slack
```

Or you download from [Releases](https://github.com/catatsuy/notify_slack/releases).

If you want to develop, please use the `make`. This software requires Go 1.12 or higher.

```
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

You post the file as a snippet. `token` and `channel` is required to use the Slack Web API.

Slack's API can specify `filetype`. You can also specify `-filetype`. But it is automatically determined from the extension of the file.
You make sure to give the appropriate extension.

[file type | Slack](https://api.slack.com/types/file#file_types)

If you want to upload to snippet via standard input, you can specify `filename`.

``` sh
git diff | ./bin/notify_slack -filename git.diff /dev/stdin
```


### CLI options

```
-c string
      config file name
-channel string
      specify channel
-filename string
      specify a file name (for uploading to snippet)
-filetype string
      specify a filetype (for uploading to snippet)
-icon-emoji string
      specify icon emoji
-interval duration
      interval (default 1s)
-slack-url string
      slack url
-token string
      token (for uploading to snippet)
-username string
      specify username
-version
      Print version information and quit
```

### toml configuration file

By default check the following files.

1. a file specified with `-c`
1. `$HOME/.notify_slack.toml`
1. `$HOME/etc/notify_slack.toml`
1. `/etc/notify_slack.toml`

The contents of the toml file are as follows.

```toml:notify_slack.toml
[slack]
url = "https://hooks.slack.com/services/**"
token = "xxxxx"
channel = "#general"
username = "tester"
icon_emoji = ":rocket:"
interval = "1s"
```

Note:

  * `url` is a required parameter.
    * You can specify `channel`, `username`, `icon_emoji` and `interval`.
  * `token` and `channel` is necessary if you want to post to snippet.
    * `username` and `icon_emoji` are ignored in this case.

Tips:

  * If you want to default to another channel only for snippet, you can use `snippet_channel`.

## Release

Version of `cli/cli.go` will be updated.

When you execute the following command and give a tag, it will be released via CircleCI.

``` sh
git tag v0.3.0
git push origin v0.3.0
```

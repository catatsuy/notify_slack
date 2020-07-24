# notify_slack

Notify slack from the command line. It receives standard input and notifies Slack all at once every second (can be changed with the `-interval` option).

Please watch this video. https://www.youtube.com/watch?v=wmKSr9Aoz-Y

## Installation

```
GO111MODULE=on go get github.com/catatsuy/notify_slack/cmd/notify_slack
```

Or you download from [Releases](https://github.com/catatsuy/notify_slack/releases).

If you want to develop, please use the `make`. This software requires Go 1.13 or higher.

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

If you want to upload to snippet via standard input, you must specify `-snippet`. If you specify `filename`, you can change the file name on Slack.

``` sh
git diff | ./bin/notify_slack -snippet -filename git.diff
```

Slack's API can specify `filetype`. You can also specify `-filetype`. But it is automatically determined from the extension of the file.
You make sure to give the appropriate extension.

[file type | Slack](https://api.slack.com/types/file#file_types)


### CLI options

```
-c string
      config file name
-channel string
      specify channel (unavailable for new Incoming Webhooks)
-filename string
      specify a file name (for uploading to snippet)
-filetype string
      specify a filetype (for uploading to snippet)
-icon-emoji string
      specify icon emoji (unavailable for new Incoming Webhooks)
-interval duration
      interval (default 1s)
-slack-url string
      slack url (Incoming Webhooks URL)
-snippet
      switch to snippet uploading mode
-token string
      token (for uploading to snippet)
-username string
      specify username (unavailable for new Incoming Webhooks)
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
token = "xoxp-xxxxx"
channel = "#general"
username = "tester"
icon_emoji = ":rocket:"
interval = "1s"
```

Note:

  * `url` is necessary if you want to post to slack as text.
    * You can specify `channel`, `username`, `icon_emoji` and `interval`.
    * Now, you cannot override `channel`, `username`, `icon_emoji` due to the specification change of Incoming Webhooks. Please refer to https://api.slack.com/messaging/webhooks#advanced_message_formatting
    * Incoming Webhooks url can be created on https://slack.com/services/new/incoming-webhook
  * `token` and `channel` is necessary if you want to post to snippet.
    * `username` and `icon_emoji` are ignored in this case.
    * Please see the next section for how to create token.

Tips:

  * If you want to default to another channel only for snippet, you can use `snippet_channel`.

### How to create a token

You need to create a token if you use snippet uploading mode.

#### Create New App

At first, you need to create new app. Please access https://api.slack.com/apps.

1. click `Create New App`
2. input application name to `App Name`
3. select your workspace on `Development Slack Workspace`
4. click `Create App`

#### Basic Information

1. click `Permissions` on `Add features and functionality`
2. select `files:write:user` on `Scopes` and click `Save Changes`. You are able to choose `Bot Token Scopes` or `User Token Scopes`

#### OAuth & Permissions

1. click `Install App to Workspace`
2. install your app
3. copy `OAuth Access Token` beginning with `xoxp-` or `Bot User OAuth Access Token` beginning with `xoxb-`

#### Add apps

1. click `Details` on the channel which you want to post
2. click `Add apps` on `More`
3. choose your app

### (Advanced) Environment Variables

Some settings can be given by the following environment variables.

```
NOTIFY_SLACK_WEBHOOK_URL
NOTIFY_SLACK_TOKEN
NOTIFY_SLACK_CHANNEL
NOTIFY_SLACK_SNIPPET_CHANNEL
NOTIFY_SLACK_USERNAME
NOTIFY_SLACK_ICON_EMOJI
```

It will be useful if you want to use it on a container. If you use it, you don't need a configuration file anymore.


## Release

When you execute the following command and give a tag, it will be released via GitHub Actions.

``` sh
git tag v0.3.0
git push origin v0.3.0
```

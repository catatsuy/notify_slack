# notify_slack

The 'notify_slack' command allows you to post messages to Slack from the command line. Simply pipe the standard output of any command to 'notify_slack', and it will send the output to Slack at a rate of once per second (this interval can be modified using the `-interval` option).

https://user-images.githubusercontent.com/1249910/155869750-48f7500f-4481-49b6-9d65-b93205f2b94f.mp4

(same movie) https://www.youtube.com/watch?v=wmKSr9Aoz-Y

## Installation

It is recommended that you use the binaries available on [GitHub Releases](https://github.com/catatsuy/notify_slack/releases). It is advisable to download and use the latest version.

If you have a Go language development environment set up, you can also compile and install the 'notify_slack' tools on your own.

```
go install github.com/catatsuy/notify_slack/cmd/notify_slack@latest
```

To build and modify the 'notify_slack' tools for development purposes, you can use the `make` command.

```
make
```

If you use the `make` command to build and install the 'notify_slack' tool, the output of the `notify_slack -version` command will be the git commit ID of the current version.

## usage

To post messages to Slack using the 'notify_slack' tool, you can either specify the necessary settings as command line options or in a TOML configuration file. If both options are provided, the command line settings will take precedence.

```sh
./bin/output | ./bin/notify_slack
```

The 'output' tool is used for testing purposes and allows you to buffer and then post messages to Slack.

``` sh
./bin/notify_slack README.md
```

To use the Slack Web API and post a file as a snippet, you will need to provide a `token` and `channel`. If you want to upload a snippet via standard input, you must specify the `-snippet` flag. You can also specify a `filename` to change the name of the file on Slack.

``` sh
git diff | ./bin/notify_slack -snippet -filename git.diff
```

The Slack API allows you to specify the filetype of a file when posting it as a snippet. You can also use the `-filetype` flag to specify the file type. If this flag is not provided, the file type will be automatically determined based on the file's extension. It is important to ensure that the extension of the file accurately reflects its type.

[file type | Slack](https://api.slack.com/types/file#file_types)


### CLI options

```
-c string
      config file name
-channel string
      specify channel (unavailable for new Incoming Webhooks)
-channel-id string
      specify channel id (for uploading a file)
-debug
      debug mode (for developers)
-filename string
      specify a file name (for uploading to snippet)
-filetype string
      [compatible] specify a filetype for uploading to snippet. This option is maintained for compatibility. Please use -snippet-type instead.
-icon-emoji string
      specify icon emoji (unavailable for new Incoming Webhooks)
-interval duration
      interval (default 1s)
-slack-url string
      slack url (Incoming Webhooks URL)
-snippet
      switch to snippet uploading mode
-snippet-type string
      specify a snippet_type (for uploading to snippet)
-token string
      token (for uploading to snippet)
-username string
      specify username (unavailable for new Incoming Webhooks)
-version
      Print version information and quit
```

### toml configuration file

By default, check the following files in order.

1. a file specified with `-c`
1. `$HOME/.notify_slack.toml`
1. `$HOME/etc/notify_slack.toml`
1. `/etc/notify_slack.toml`

The toml file contains the following information.

```toml:notify_slack.toml
[slack]
url = "https://hooks.slack.com/services/**"
token = "xoxp-xxxxx"
channel = "#general"
channel_id = "C12345678"
username = "tester"
icon_emoji = ":rocket:"
interval = "1s"
```

### Note

  * You will need to specify a url if you want to post messages to Slack as text
    * You can use the following options to customize your message when posting to Slack as text: `channel`, `username`, `icon_emoji`, and `interval`.
    * Due to a recent change in the specification for Incoming Webhooks, it is currently not possible to override the `channel`, `username`, and `icon_emoji` options when posting to Slack. For more information, please refer to [this resource](https://api.slack.com/messaging/webhooks#advanced_message_formatting)
    * You can create an Incoming Webhooks URL at https://slack.com/services/new/incoming-webhook
  * To post a file as a snippet to Slack, you will need to provide both a `token` and a `channel_id`.
    * The `username` and `icon_emoji` options will be ignored when posting a file as a snippet to Slack.
    * For instructions on how to create a token, please see the next section.
    * You cannot specify a channel because the slack api support only the `channel_id`.
    * If you don't specify `channel_id`, the file will be private. So, **if you need to post a file public, you must specify `channel_id`**.
    * The Slack API can cause delays, so posting might take longer.

### Getting Your Slack API Token

You need to create a token if you use snippet uploading mode.

For the most up-to-date and easy-to-follow instructions on how to obtain your Slack API bot token, please refer to the official Slack guide:

[How to quickly get and use a Slack API bot token | Slack](https://api.slack.com/tutorials/tracks/getting-a-token)

### (Advanced) Environment Variables

Some settings for the Slack API can be provided using environment variables.

```
NOTIFY_SLACK_WEBHOOK_URL
NOTIFY_SLACK_TOKEN
NOTIFY_SLACK_CHANNEL
NOTIFY_SLACK_CHANNEL_ID
NOTIFY_SLACK_USERNAME
NOTIFY_SLACK_ICON_EMOJI
NOTIFY_SLACK_INTERVAL
```

Using environment variables to specify settings for the 'notify_slack' tool can be useful if you are deploying it in a containerized environment. It allows you to avoid the need for a configuration file and simplifies the process of managing and updating settings.

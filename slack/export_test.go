package slack

func SetSlackFilesUploadURL(u string) (resetFunc func()) {
	var tmp string
	tmp, slackFilesUploadURL = slackFilesUploadURL, u
	return func() {
		slackFilesUploadURL = tmp
	}
}

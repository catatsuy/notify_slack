package slack

func SetSlackFilesUploadURL(u string) (resetFunc func()) {
	var tmp string
	tmp, slackFilesUploadURL = slackFilesUploadURL, u
	return func() {
		slackFilesUploadURL = tmp
	}
}

func SetFilesGetUploadURLExternalURL(u string) (resetFunc func()) {
	var tmp string
	tmp, filesGetUploadURLExternalURL = filesGetUploadURLExternalURL, u
	return func() {
		filesGetUploadURLExternalURL = tmp
	}
}

func SetFilesCompleteUploadExternalURL(u string) (resetFunc func()) {
	var tmp string
	tmp, filesCompleteUploadExternalURL = filesCompleteUploadExternalURL, u
	return func() {
		filesCompleteUploadExternalURL = tmp
	}
}

package slack

func SetFilesGetUploadURLExternalURL(u string) (resetFunc func()) {
	var tmp string
	tmp, filesGetUploadURLExternalURL = filesGetUploadURLExternalURL, u
	return func() {
		filesGetUploadURLExternalURL = tmp
	}
}

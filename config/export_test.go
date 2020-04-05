package config

func SetUserHomeDir(u string) (resetFunc func()) {
	tmp := userHomeDir
	userHomeDir = func() (string, error) {
		return u, nil
	}
	return func() {
		userHomeDir = tmp
	}
}

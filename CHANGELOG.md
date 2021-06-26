# Change Log

## [v0.4.11] - 2021-06-26

### Changed

* update Go version

## [v0.4.10] - 2021-03-28

### Changed

* Require Go 1.16

## [v0.4.9] - 2021-02-21

### Added

* Added support for `NOTIFY_SLACK_INTERVAL` environment variables
* Added the binary for Mac ARM64

### Changed

* refactoring

## [v0.4.8] - 2020-12-19

### Changed

* Changed version number is included by `go get`

## [v0.4.7] - 2020-07-26

### Added

* Added the binary for Linux ARM64
* Modify README due to specification changes by Slack

## [v0.4.6] - 2020-04-05

### Changed

* Changed to use Go 1.14
* Changed a test

## [v0.4.5] - 2020-02-29

### Changed

* Changed to use GoReleaser on GitHub Actions

## [v0.4.4] - 2020-02-02

### Changed

* Changed to use GitHub Actions instead of Circle CI

## [v0.4.3] - 2019-11-01

### Changed

* Changed `url` not to be a required parameter
* Modified README

## [v0.4.2] - 2019-10-30

### Changed

* Reduced the number of upgrade steps
* Reduced the number of dependent libraries

## [v0.4.1] - 2019-07-03

### Fixed

* Fixed a bug that caused permission denied error #43 @ryotarai

## [v0.4.0] - 2019-06-22

### Added

* Added `-snippet` option
* Added support for `NOTIFY_SLACK_*` environment variables
* Added `~/.notify_slack.toml` as a configuration file

## [v0.3.0] - 2019-03-09

### Added

* Added automatically release file by CircleCI

### Changed

* Changed to use Go 1.12
* Modified using Go modules

## [v0.2.1] - 2018-09-03

### Added

* Add version output function
* Added support passing interval from toml file

## [v0.2] - 2018-09-01

### Added

* Added support for uploading to snippet

## [v0.1] - 2017-09-24

* First release

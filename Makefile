export GO111MODULE=on

.PHONY: all test vet release

all: bin/notify_slack bin/output

bin/notify_slack: cmd/notify_slack/main.go slack/*.go throttle/*.go config/*.go cli/*.go
	go build -o bin/notify_slack cmd/notify_slack/main.go

bin/output: cmd/output/main.go
	go build -o bin/output cmd/output/main.go

release:
	GOOS=linux go build -o notify_slack cmd/notify_slack/main.go
	tar cvzf release/notify_slack-linux-amd64.tar.gz notify_slack
	GOOS=darwin go build -o notify_slack cmd/notify_slack/main.go
	tar cvzf release/notify_slack-darwin-amd64.tar.gz notify_slack
	rm notify_slack

test:
	go test -count 10 ./...

vet:
	go vet ./...

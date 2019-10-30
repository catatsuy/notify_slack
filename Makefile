export GO111MODULE=on

.PHONY: all
all: bin/notify_slack bin/output

go.mod go.sum:
	go mod tidy

bin/notify_slack: cmd/notify_slack/main.go slack/*.go throttle/*.go config/*.go cli/*.go go.mod go.sum
	go build -o bin/notify_slack cmd/notify_slack/main.go

bin/output: cmd/output/main.go
	go build -o bin/output cmd/output/main.go

.PHONY: release
release:
	GOOS=linux go build -ldflags "-X github.com/catatsuy/notify_slack/cli.Version=${version}" -o notify_slack cmd/notify_slack/main.go
	tar cvzf release/notify_slack-linux-amd64.tar.gz notify_slack
	GOOS=darwin go build -ldflags "-X github.com/catatsuy/notify_slack/cli.Version=${version}" -o notify_slack cmd/notify_slack/main.go
	tar cvzf release/notify_slack-darwin-amd64.tar.gz notify_slack
	rm notify_slack

.PHONY: test
test:
	go test -count 10 ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: errcheck
errcheck:
	errcheck ./...

.PHONY: staticcheck
staticcheck:
	staticcheck -checks="all,-ST1000" ./...

.PHONY: clean
clean:
	rm -rf bin/*

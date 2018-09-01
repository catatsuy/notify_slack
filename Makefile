GLIDE = glide

.PHONY: all test vet bundle release

all: bin/notify_slack bin/output

bin/notify_slack: cmd/notify_slack/main.go slack/*.go throttle/*.go config/*.go cli/*.go
	go build -o bin/notify_slack cmd/notify_slack/main.go

bin/output: cmd/output/main.go
	go build -o bin/output cmd/output/main.go

release:
	GOOS=linux go build -o notify_slack cmd/notify_slack/main.go
	tar cvf release/notify_slack-linux-amd64.tar.gz notify_slack
	GOOS=darwin go build -o notify_slack cmd/notify_slack/main.go
	tar cvf release/notify_slack-darwin-amd64.tar.gz notify_slack
	rm notify_slack

bundle:
	$(GLIDE) install

test:
	go test ./...

vet:
	go vet ./...

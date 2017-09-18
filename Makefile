.PHONY: test

bin/notify_slack: cmd/notify_slack/main.go slack/*.go throttle/*.go
	go build -o bin/notify_slack cmd/notify_slack/main.go

test:
	go test ./...

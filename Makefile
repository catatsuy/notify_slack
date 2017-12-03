GLIDE = glide

.PHONY: all test bundle

all: bin/notify_slack bin/output

bin/notify_slack: cmd/notify_slack/main.go slack/*.go throttle/*.go config/*.go
	go build -o bin/notify_slack cmd/notify_slack/main.go

bin/output: cmd/output/main.go
	go build -o bin/output cmd/output/main.go

bundle:
	$(GLIDE) install

test:
	go test ./...

vet:
	go vet ./...

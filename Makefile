.PHONY: all
all: bin/notify_slack bin/output

go.mod go.sum:
	go mod tidy

bin/notify_slack: cmd/notify_slack/main.go internal/slack/*.go internal/throttle/*.go internal/config/*.go internal/cli/*.go go.mod go.sum
	go build -ldflags "-X github.com/catatsuy/notify_slack/internal/cli.Version=`git rev-list HEAD -n1`" -o bin/notify_slack cmd/notify_slack/main.go

bin/output: cmd/output/main.go
	go build -o bin/output cmd/output/main.go

.PHONY: test
test:
	go test -shuffle on -cover -count 10 ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: errcheck
errcheck:
	errcheck ./...

.PHONY: staticcheck
staticcheck:
	staticcheck -checks="all,-ST1000" ./...

.PHONY: cover
cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: clean
clean:
	rm -rf bin/*

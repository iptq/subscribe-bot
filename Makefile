GIT_COMMIT = $(shell git rev-list -1 HEAD)
SOURCES = $(shell find . -type f -and -name "*.go" -or -name "*.html")

all: subscribe-bot

subscribe-bot: $(SOURCES)
	go build -o $@ -ldflags "-X main.GitCommit=$(GIT_COMMIT)"

.PHONY: all
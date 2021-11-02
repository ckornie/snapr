GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GORUN=$(GOCMD) run
BUILD_DIR=build
TARGET=cmd/snapr/main.go

all: build

build:
		mkdir -p $(BUILD_DIR)
		$(GOBUILD) -o $(BUILD_DIR) -v ./...

test:
		$(GOTEST) -count=1 -v ./...

clean:
		$(GOCLEAN)
		rm -f $(BUILD_DIR)/*

run:
		$(GORUN) $(TARGET)

build-linux:
		mkdir -p $(BUILD_DIR)
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags="-s -w" -o $(BUILD_DIR) -v ./...

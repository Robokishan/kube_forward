# Makefile for building kube_forward.go

BINARY_NAME = kube_forward
SRC = kube_forward.go

.PHONY: all build clean

all: build

build:
	go build -o $(BINARY_NAME) $(SRC)

clean:
	rm -f $(BINARY_NAME)
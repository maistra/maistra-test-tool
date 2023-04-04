#!/bin/bash

go version
golangci-lint version

golangci-lint run -v -c ./scripts/.golangci.yaml

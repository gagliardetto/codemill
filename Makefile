.DEFAULT_GOAL := install
install:
	go build -o $$GOPATH/bin/codemill
run:
	GOPACKAGESDEBUG=true GO111MODULE=on GOOS=linux GOARCH=amd64 go run main.go

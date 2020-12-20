.DEFAULT_GOAL := install

ifeq ($(http),)
http := "true"
endif

ifeq ($(gen),)
gen := "true"
endif

install:
	go build -o $$GOPATH/bin/codemill
run:
	GOPACKAGESDEBUG=true GO111MODULE=on GOOS=linux GOARCH=amd64 go run main.go --spec=$(spec) --dir=$(dir) --http=$(http) --gen=$(gen)

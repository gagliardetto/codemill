.DEFAULT_GOAL := install

ifeq ($(http),)
http := "true"
endif

ifeq ($(gen),)
gen := "false"
endif

generate:
	go generate
install: generate
	go build -o $$GOPATH/bin/codemill
run: generate
	GOPACKAGESDEBUG=true GO111MODULE=on GOOS=linux GOARCH=amd64 go run main.go --spec=$(spec) --dir=$(dir) --http=$(http) --gen=$(gen)

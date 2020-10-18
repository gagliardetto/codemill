https://blog.golang.org/module-mirror-launch
https://github.com/golang/gofrontend/blob/2c390ba951e83b547f6387cc9e19436c085b3775/libgo/go/cmd/go/internal/modload/import.go#L113
https://github.com/golang/go/blob/c9211577eb77df9c51f0565f1da7d20ff91d59df/src/cmd/go/internal/modfetch/fetch.go
https://github.com/golang/go/blob/846dce9d05f19a1f53465e62a304dea21b99f910/src/cmd/go/internal/modload/query.go#L269
https://github.com/golang/go/blob/846dce9d05f19a1f53465e62a304dea21b99f910/src/cmd/go/internal/modload/query.go#L269

https://github.com/golang/tools/blob/f1b4bd93c9465ac3d4edf2a53caf28cd21f846aa/go/ssa/example_test.go

find repo:
	https://api.godoc.org/search?q=godoc

get a list of versions:
	https://proxy.golang.org/github.com/gin-gonic/gin/@v/list

if no version, get latest version:
	https://proxy.golang.org/github.com/gagliardetto/codebox/@latest

fetch all code (NOTE: there are steps before that, which I don't understand yet)
	https://proxy.golang.org/github.com/gin-gonic/gin/@v/v1.3.0.zip





---

```
find . -name '*.go' -exec sed -i -e 's/"cmd\//"github.com\/gagliardetto\/codemill\/cmd\//g' {} \;
find . -name '*.go' -exec sed -i -e 's/"github.com\/gagliardetto\/codemill\/cmd\/go\/internal/"github.com\/gagliardetto\/codemill\/cmd\/go\/not-internal/g' {} \;
find . -name '*.go' -exec sed -i -e 's/"internal\//"github.com\/gagliardetto\/codemill\/not-internal\//g' {} \;
```

---

export GOPRIVATE=github.com/gagliardetto/gomill/\* 
go env -w GOPRIVATE=github.com/<OrgNameHere>/*

GO111MODULE=on GOOS=linux GOARCH=amd64 go run main.go
### Flow of adding a new library (framework, etc.)

- run codemill
- Copy .qll to ~/vscode-codeql-starter/codeql-go/ql/src/semmle/go/frameworks
- Copy tests folder to ~/vscode-codeql-starter/codeql-go/ql/test/library-tests/semmle/go/frameworks
- Add import to ~/vscode-codeql-starter/codeql-go/ql/src/go.qll
	- Same paths? Add a `packagePath` predicate.
	- Same versions and vendor across all models? Move /vendor and `go.mod` to parent dir of tests.
- cd testdir
- run go generate
- run gn (i.e. gonull .)
- run codeql tests


### Run a codeql test

```bash
~/codeql-home/codeql-cli-v.2.4.1/codeql test run \
        --search-path=~/codeql-home/codeql-cli-v.2.4.1 \
        --search-path=~/vscode-codeql-starter/codeql-go/ql \
        ~/vscode-codeql-starter/codeql-go/ql/test/library-tests/semmle/go/frameworks/CleverGo/UntrustedSources
```

### Notes

NOTE: untrusteflowsource does not work with string/int types passed as parameters and then sinked.
Example:
```golang
{
	var paramStrs414 string
	result518 := revel.FirstNonEmpty(paramStrs414)
	sink(paramStrs414) // $SinkingUntrustedFlowSource
}
```
- Cannot flow *into* a string parameter.




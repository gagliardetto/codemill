~/codeql-home/codeql-cli-v.2.4.1/codeql test run \
        --search-path=~/codeql-home/codeql-cli-v.2.4.1 \
        --search-path=~/vscode-codeql-starter/codeql-go/ql \
        ~/vscode-codeql-starter/codeql-go/ql/test/library-tests/semmle/go/frameworks/CleverGo/UntrustedSources


- run codemill
- Copy .qll to ~/vscode-codeql-starter/codeql-go/ql/src/semmle/go/frameworks
- Add import to ~/vscode-codeql-starter/codeql-go/ql/src/go.qll
- Copy tests folder to ~/vscode-codeql-starter/codeql-go/ql/test/library-tests/semmle/go/frameworks
- cd testdir
- run go generate
- run gn (i.e. gonull .)
- run codeql tests


NOTE: untrusteflowsource does not work with string/int types passed as parameters and then sinked.
Example:
{
	var paramStrs414 string
	result518 := revel.FirstNonEmpty(paramStrs414)
	sink(paramStrs414) // $SinkingUntrustedFlowSource
}
- Cannot flow *into* a string parameter.




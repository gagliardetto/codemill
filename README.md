## codemill

`codemill` helps with the creation of codeql models for Go.

You build a spec of a module in a browser-based UI, adding models and selectors to it, and then it generates the corresponding codeql and go code.


### Currently supported codeql models

- **TaintTracking** - DONE
- **UntrustedFlowSource** - DONE
- **HTTP::HeaderWrite** - WIP
- **HTTP::Redirect** - WIP
- **HTTP::ResponseBody** - WIP

## Install

You can install codemill cloning this repo and running `make install`; you need Go < 1.16.

**NOTE**: Go 1.16 will be supported soon (WIP).

## Example generated code

You can see an example of what `codemill` can generate here: https://github.com/github/codeql-go/pull/438/files

## Example: gin

```bash
# Welcome to a `codemill` basic usage example
# First' let's create a folder for our codemill files
mkdir my-codemill && cd my-codemill
# Then we need a folder for the projects' specs
mkdir specs
# And a folder for generated files
mkdir generated
# Now we're ready for creating our first spec
# In this example I will create a very incomplete model for the gin web framework
codemill --spec=./specs/Gin.json --dir=./generated --http=true --gen=true
```

![codemill-initial-setup](https://user-images.githubusercontent.com/15271561/109022902-f326b580-76c4-11eb-856c-4969ea5f80d3.gif)

After that, let's open [http://127.0.0.1:8070/](http://127.0.0.1:8070/) in a browser, and edit the spec.

The first model we will add to the `Gin` spec is an `UntrustedFlowSource` model, which defines sources of user-defined input:

![codemill-gin-untrustedflowsource](https://user-images.githubusercontent.com/15271561/113894302-8aa51b00-97d0-11eb-9645-c2a4f1d26958.gif)

The second model we will add to the `Gin` spec is a `TaintTracking` model, which defines taint propagation in functions and methods:

![codemill-gin-tainttracking](https://user-images.githubusercontent.com/15271561/109023904-db9bfc80-76c5-11eb-9449-f264bc3b8886.gif)

Now our spec is done, let's go back to the terminal and hit `CTRL+C` to close the program.

On exit, `codemill` will save the `Gin` spec we just created to `specs/Gin.json`, and generate codeql and go files in a timestamped folder inside the `generated/` folder.

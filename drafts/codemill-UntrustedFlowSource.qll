/**
 * This is a doc.
 * Second line of doc.
 */

import go
import DataFlow::PathGraph

/**
 * Doc about this module.
 */
private module SomeFramework {
  /**
   * Doc about class
   */
  private class UntrustedSource extends UntrustedFlowSource::Range {
    UntrustedSource() {
      // Example block 1: the type is a function.
      // The source is either the result (one or more), or a parameter (one or more),
      // or a mix of them.
      (
        // Function: github.com/example/package.GetSomething
        exists(Function fn, DataFlow::CallNode call |
          fn.hasQualifiedName("github.com/example/package", "GetSomething")
        |
          // The source is the result:
          call = fn.getACall() and this = call.getResult()
        )
        or
        // Function: github.com/example/package.ParseSomething
        exists(Function fn, DataFlow::CallNode call |
          fn.hasQualifiedName("github.com/example/package", "ParseSomething")
        |
          // The source is the 0th parameter:
          call = fn.getACall() and this = FunctionOutput::parameter(0).getExitNode(call)
        )
      )
      or
      // Example block 2: the type is a struct.
      // The source can be a method call (results or parameters of a call),
      // or a field read.
      (
        // Struct: github.com/example/package.Context
        exists(string typeName | typeName = "Context" |
          // Method calls on `Context`:
          exists(DataFlow::MethodCallNode call, string methodName |
            call.getTarget().hasQualifiedName("github.com/example/package", typeName, methodName) and
            (
              methodName = "FullPath" and
              (
                // The source is the method call result #0:
                this = call.getResult(0)
                or
                // The source is method call parameter #0:
                this = FunctionOutput::parameter(0).getExitNode(call)
              )
              or
              // The source is any result of the call?
              methodName = "GetHeader"
            )
          )
          or
          // Field reads on `Context`:
          exists(DataFlow::Field fld, string fieldName |
            fld.hasQualifiedName("github.com/example/package", typeName, fieldName) and
            // The source is one of these fields reads:
            fieldName in ["Accepted", "Params"] and
            this = fld.getARead()
          )
        )
        or
        // Struct: github.com/example/package.SomeStruct
        exists(DataFlow::Field fld, string fieldName |
          fld.hasQualifiedName("github.com/example/package", "SomeStruct", fieldName) and
          // The source is one of these fields reads:
          fieldName in ["Hello", "World"] and
          this = fld.getARead()
        )
      )
      or
      // Example block 3: the type is some custom type.
      // The source is a method call on that type (results or parameters of a call).
      // Some type: github.com/example/package.Slice
      exists(string typeName | typeName = "Slice" |
        // Method calls:
        exists(DataFlow::MethodCallNode call, string methodName |
          call.getTarget().hasQualifiedName("github.com/example/package", typeName, methodName) and
          (
            methodName = "GetHeader" and
            // The source is the method call result #0:
            this = call.getResult(0)
            or
            methodName = "ParseHeader" and
            // The source is method call parameter #1:
            this = FunctionOutput::parameter(1).getExitNode(call)
          )
        )
      )
      or
      // Example block 4: the type is an interface.
      // The source is a method call on that interface (results or parameters of a call).
      // Interface: github.com/example/package.SomeInterface
      exists(string typeName | typeName = "SomeInterface" |
        // Method calls:
        exists(DataFlow::MethodCallNode call, string methodName |
          call.getTarget().implements("github.com/example/package", typeName, methodName) and
          (
            methodName = "GetSomething" and
            // The source is the method call result #0:
            this = call.getResult(0)
            or
            methodName = "UnmarshalSomething" and
            // The source is method call parameter #2:
            this = FunctionOutput::parameter(2).getExitNode(call)
          )
        )
      )
      or
      // Example block 5: the type is whatever.
      // The source is a read of a variable of that type.
      // Type: github.com/example/package.Slice
      exists(DataFlow::ReadNode read, ValueEntity v |
        read.reads(v) and
        v.getType().hasQualifiedName("github.com/example/package", "Slice")
      |
        this = read
      )
    }
  }
}

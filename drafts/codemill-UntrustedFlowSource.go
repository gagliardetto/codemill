package main

type Type struct {
}

//
func (n *Type) Method(param string) string {
	return ""
}

func ThisIsAFunc(param string) string {
	return ""
}

func sink(p interface{}) {

}
func main() {
	{ // Func WITHOUT a receiver:
		{
			// Parameter:
			var param string
			ThisIsAFunc(param)
			sink(param)
		}

		{
			// Result:
			result := ThisIsAFunc("")
			sink(result)
		}
	}
	{ // Func with a receiver:
		{
			// Receiver:
			var t Type
			t.Method("")
			sink(t)
		}

		{
			// Parameter:
			var t Type
			var param string

			t.Method(param)
			sink(param)
		}

		{
			// Result:
			var t Type
			result := t.Method("")
			sink(result)
		}
	}

	// Struct:
	{

	}

}

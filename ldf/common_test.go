package ldf_test

import "fmt"

type LdfBool bool

func (b LdfBool) Format(f fmt.State, verb rune) {
	switch verb {
	case 't':
		out := "0"
		if b {
			out = "1"
		}
		f.Write([]byte(out))
	default:
		fmt.Fprintf(f, fmt.FormatString(f, verb), bool(b))
	}
}

type Basic struct {
	String  string  `ldf:"STRING"`
	Int32   int32   `ldf:"INT32"`
	Float   float32 `ldf:"FLOAT"`
	Double  float64 `ldf:"DOUBLE"`
	Uint32  uint32  `ldf:"UINT32"`
	Boolean LdfBool `ldf:"BOOLEAN"`
}

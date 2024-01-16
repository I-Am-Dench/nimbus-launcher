package ldf_test

type Basic struct {
	String  string  `ldf:"STRING"`
	Int32   int32   `ldf:"INT32"`
	Float   float32 `ldf:"FLOAT"`
	Double  float64 `ldf:"DOUBLE"`
	Uint32  uint32  `ldf:"UINT32"`
	Boolean bool    `ldf:"BOOLEAN"`
}

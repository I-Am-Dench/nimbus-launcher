package ldf_test

import (
	"fmt"
	"testing"

	"github.com/I-Am-Dench/nimbus-launcher/ldf"
)

func hasSameContents(a, b []byte) (bool, int, error) {
	if len(a) != len(b) {
		return false, 0, fmt.Errorf("len(a) != len(b); %d != %d", len(a), len(b))
	}

	for i, v := range a {
		if v != b[i] {
			return false, i, fmt.Errorf("mismatched index: (a[%d] = %d) != (b[%d] = %d)", i, v, i, b[i])
		}
	}

	return true, 0, nil
}

func TestMarshal(t *testing.T) {
	v := Basic{
		String:  "If just being born is the greatest act of creation. Then what are you suppose to do after that? Isn't everything that comes next just sort of a disappointment? Slowly entropying until we deflate into a pile of mush?",
		Int32:   2123311855,
		Float:   0.2394242421,
		Double:  -15555313.199119,
		Uint32:  2340432028,
		Boolean: true,
	}

	b := 0
	if v.Boolean {
		b = 1
	}
	expectedString := fmt.Sprintf("STRING=0:%s,INT32=1:%d,FLOAT=3:%v,DOUBLE=4:%v,UINT32=5:%d,BOOLEAN=7:%d", v.String, v.Int32, v.Float, v.Double, v.Uint32, b)

	data, err := ldf.Marshal(v)
	if err != nil {
		t.Fatalf("test marshal: %v", err)
	}

	if ok, index, err := hasSameContents(data, []byte(expectedString)); !ok {
		t.Errorf("test unmarshal: %v", err)
		t.Fatalf("expected =\n%v\nactual =\n%v\n", string(data[index:]), expectedString[index:])
	}
}

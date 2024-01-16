package ldf_test

import (
	"fmt"
	"testing"

	"github.com/I-Am-Dench/lu-launcher/ldf"
)

func TestUnmarshal(t *testing.T) {
	expectedString := "Save Imagination! :)"
	expectedInt32 := int32(2010)
	expectedFloat := float32(39.99)
	expectedDouble := float64(3.14159265358932384)
	expectedUint32 := uint32(4051612861)

	data := []byte(fmt.Sprintf("STRING=0:%s,INT32=1:%d,FLOAT=3:%f,DOUBLE=4:%v,UINT32=5:%d,BOOLEAN=7:1", expectedString, expectedInt32, expectedFloat, expectedDouble, expectedUint32))

	v := Basic{}
	err := ldf.Unmarshal(data, &v)
	if err != nil {
		t.Fatalf("test unmarshal: %v", err)
	}

	if v.String != expectedString {
		t.Fatalf("test unmarshal: expected string \"%s\" but got %s", expectedString, v.String)
	}

	if v.Int32 != expectedInt32 {
		t.Fatalf("test unmarshal: expected int32 %d but got %d", expectedInt32, v.Int32)
	}

	if v.Float != expectedFloat {
		t.Fatalf("test unmarshal: expected float %f but got %f", expectedFloat, v.Float)
	}

	if v.Double != expectedDouble {
		t.Fatalf("test unmarshal: expected double %g but got %g", expectedDouble, v.Double)
	}

	if v.Uint32 != expectedUint32 {
		t.Fatalf("test unmarshal: expected uint32 %d but got %d", expectedUint32, v.Uint32)
	}

	if !v.Boolean {
		t.Fatalf("test unmarshal: expected true but got %t", v.Boolean)
	}
}

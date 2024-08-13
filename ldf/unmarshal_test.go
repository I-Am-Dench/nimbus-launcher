package ldf_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/I-Am-Dench/nimbus-launcher/ldf"
)

var (
	formatCommasOnly = "STRING=0:%s,INT32=1:%d,FLOAT=3:%f,DOUBLE=4:%v,UINT32=5:%d,BOOLEAN=7:%t"

	formatNewlines = `
STRING=0:%s
INT32=1:%d
FLOAT=3:%f
DOUBLE=4:%v
UINT32=5:%d
BOOLEAN=7:%t`

	formatWhitespace = `

	STRING=0:%s



INT32=1:%d
FLOAT=3:%f


DOUBLE=4:%v
UINT32=5:%d

      BOOLEAN=7:%t
	`

	formatMixedCommasAndNewlines = `STRING=0:%s
INT32=1:%d,
FLOAT=3:%f
DOUBLE=4:%v
UINT32=5:%d,
BOOLEAN=7:%t`

	formatCarriageReturns = "STRING=0:%s\r\nINT32=1:%d\r\nFLOAT=3:%f\r\nDOUBLE=4:%v\r\nUINT32=5:%d\r\nBOOLEAN=7:%t"
)

func testUnmarshalBasic(expected Basic, format string) error {
	data := []byte(fmt.Sprintf(format, expected.String, expected.Int32, expected.Float, expected.Double, expected.Uint32, expected.Boolean))

	actual := Basic{}
	if err := ldf.Unmarshal(data, &actual); err != nil {
		return err
	}

	errs := []error{}

	if actual.String != expected.String {
		errs = append(errs, fmt.Errorf("expected string \"%s\" but got %s", expected.String, actual.String))
	}

	if actual.Int32 != expected.Int32 {
		errs = append(errs, fmt.Errorf("expected int32 %d but got %d", expected.Int32, actual.Int32))
	}

	if actual.Float != expected.Float {
		errs = append(errs, fmt.Errorf("expected float %f but got %f", expected.Float, actual.Float))
	}

	if actual.Double != expected.Double {
		errs = append(errs, fmt.Errorf("expected double %g but got %g", expected.Double, actual.Double))
	}

	if actual.Uint32 != expected.Uint32 {
		errs = append(errs, fmt.Errorf("expected uint32 %d but got %d", expected.Uint32, actual.Uint32))
	}

	if actual.Boolean != expected.Boolean {
		errs = append(errs, fmt.Errorf("expected %t but got %t", expected.Boolean, actual.Boolean))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func TestUnmarshal(t *testing.T) {
	expected := Basic{
		String:  "Save Imagination! :)",
		Int32:   int32(2010),
		Float:   float32(39.99),
		Double:  float64(3.14159265358932384),
		Uint32:  uint32(4051612861),
		Boolean: true,
	}

	if err := testUnmarshalBasic(expected, formatCommasOnly); err != nil {
		t.Errorf("test unmarshal commas only: %v", err)
	}

	if err := testUnmarshalBasic(expected, formatNewlines); err != nil {
		t.Errorf("test unmarshal newlines only: %v", err)
	}

	if err := testUnmarshalBasic(expected, formatWhitespace); err != nil {
		t.Errorf("test unmarshal whitespace: %v", err)
	}

	if err := testUnmarshalBasic(expected, formatMixedCommasAndNewlines); err != nil {
		t.Errorf("test unmarshal mixed commas and newlines: %v", err)
	}

	if err := testUnmarshalBasic(expected, formatCarriageReturns); err != nil {
		t.Errorf("test unmarshal carriage returns: %v", err)
	}
}

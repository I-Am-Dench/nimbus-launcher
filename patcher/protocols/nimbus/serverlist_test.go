package nimbus_test

import (
	"bytes"
	"encoding/xml"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/I-Am-Dench/goverbuild/encoding/ldf"
	"github.com/I-Am-Dench/nimbus-launcher/patcher/protocols/nimbus"
)

func TestLdfEntries(t *testing.T) {
	data := `
	<Data>
		<Config>
			<string id="STRING">Sample String</string>
			<utf8 id="UTF8">Utf8 data</utf8>
			<int32 id="INT32">396456</int32>
			<uint32 id="UINT32">824510</uint32>
			<int64 id="INT64">-208526</int64>
			<uint64 id="UINT64">277886</uint64>
			<float32 id="FLOAT32">16.08430721</float32>
			<float64 id="FLOAT64">378.62259122</float64>
			<bool id="BOOL">1</bool>
		</Config>
	</Data>
	`

	type Data struct {
		XMLName xml.Name          `xml:"Data"`
		Config  nimbus.LdfEntries `xml:"Config"`
	}
	expected := nimbus.LdfEntries{
		{Key: "STRING", Value: "Sample String"},
		{Key: "UTF8", Value: []byte("Utf8 data")},
		{Key: "INT32", Value: int32(396456)},
		{Key: "UINT32", Value: uint32(824510)},
		{Key: "INT64", Value: int64(-208526)},
		{Key: "UINT64", Value: uint64(277886)},
		{Key: "FLOAT32", Value: float32(16.08430721)},
		{Key: "FLOAT64", Value: float64(378.62259122)},
		{Key: "BOOL", Value: true},
	}

	actual := Data{}
	if err := xml.Unmarshal([]byte(data), &actual); err != nil {
		t.Fatal(err)
	}

	if len(expected) != len(actual.Config) {
		t.Fatalf("expected %d entries but got %d", len(expected), len(actual.Config))
	}

	slices.SortFunc(expected, func(a, b ldf.Entry) int { return strings.Compare(a.Key, b.Key) })
	slices.SortFunc(actual.Config, func(a, b ldf.Entry) int { return strings.Compare(a.Key, b.Key) })

	for i, a := range expected {
		b := actual.Config[i]

		if a.Key != b.Key {
			t.Errorf("expected key %s but got %s", a.Key, b.Key)
		}

		if reflect.TypeOf(a.Value) != reflect.TypeOf(b.Value) {
			t.Errorf("%s: expected type %T but got %T", a.Key, a.Value, b.Value)
			continue
		}

		switch v := a.Value.(type) {
		case []byte:
			if !bytes.Equal(v, b.Value.([]byte)) {
				t.Errorf("%s: expected %v but got %v", a.Key, a.Value, b.Value)
			}
		default:
			if a.Value != b.Value {
				t.Errorf("%s: expected %v but got %v", a.Key, a.Value, b.Value)
			}
		}
	}
}

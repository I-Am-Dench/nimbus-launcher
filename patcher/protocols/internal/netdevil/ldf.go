package netdevil

import (
	"encoding/xml"
	"errors"
	"io"

	"github.com/I-Am-Dench/goverbuild/encoding/ldf"
)

type LdfEntries []ldf.Entry

func (l LdfEntries) Map() ldf.Map {
	m := ldf.Map{}
	for _, entry := range l {
		m[entry.Key] = entry.Value
	}
	return m
}

func (l *LdfEntries) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	entries := []ldf.Entry{}

	type keyValue struct {
		XMLName xml.Name
		Id      string `xml:"id,attr"`
		Value   []byte `xml:",chardata"`
	}

	toToken := func(kv keyValue) (ldf.Token, bool) {
		var t ldf.ValueType
		switch kv.XMLName.Local {
		case "string":
			t = ldf.ValueTypeString
		case "utf8":
			t = ldf.ValueTypeUtf8
		case "int32":
			t = ldf.ValueTypeI32
		case "uint32":
			t = ldf.ValueTypeU32
		case "int64":
			t = ldf.ValueTypeI64
		case "uint64":
			t = ldf.ValueTypeU64
		case "float32":
			t = ldf.ValueTypeFloat
		case "float64":
			t = ldf.ValueTypeDouble
		case "bool":
			t = ldf.ValueTypeBool
		default:
			return ldf.Token{}, false
		}

		return ldf.Token{
			Key:   kv.Id,
			Type:  t,
			Value: kv.Value,
		}, true
	}

loop:
	for {
		t, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return err
		}

		switch token := t.(type) {
		case xml.StartElement:
			var kv keyValue
			err := d.DecodeElement(&kv, &token)
			if errors.Is(err, io.EOF) {
				break loop
			}

			if err != nil {
				return err
			}

			tok, ok := toToken(kv)
			if !ok {
				continue
			}

			entry, err := tok.Entry()
			if err != nil {
				return err
			}

			entries = append(entries, entry)
		case xml.EndElement:
			if token.Name.Local == start.Name.Local {
				break
			}
		}
	}
	*l = entries

	return nil
}

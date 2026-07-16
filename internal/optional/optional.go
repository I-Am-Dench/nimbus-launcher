package optional

import (
	"encoding/json"
	"encoding/xml"
)

var (
	_ xml.Unmarshaler  = (*O[any])(nil)
	_ xml.Marshaler    = (*O[any])(nil)
	_ json.Unmarshaler = (*O[any])(nil)
	_ json.Marshaler   = (*O[any])(nil)
)

type O[T any] struct {
	valid bool
	Value T
}

func (o O[T]) HasValue() bool {
	return o.valid
}

func (o *O[T]) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var v *T
	if err := d.DecodeElement(&v, &start); err != nil {
		return err
	}

	if v != nil {
		o.valid = true
		o.Value = *v
	}
	return nil
}

func (o O[T]) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if o.valid {
		return e.EncodeElement(o.Value, start)
	}
	return e.EncodeElement("null", start)
}

func (o *O[T]) UnmarshalJSON(b []byte) error {
	var v *T
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	if v != nil {
		o.valid = true
		o.Value = *v
	}
	return nil
}

func (o O[T]) MarshalJSON() ([]byte, error) {
	if o.valid {
		return json.Marshal(o.Value)
	}
	return []byte("null"), nil
}

func From[T any](v T) O[T] {
	return O[T]{valid: true, Value: v}
}

package netdevil

import (
	"encoding/xml"

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
	type KeyValue struct {
		Id    string `xml:"id,attr"`
		Value []byte `xml:",chardata"`
	}

	appendEntries := func(entries *[]ldf.Entry, kvs []KeyValue, valueType ldf.ValueType) error {
		for _, kv := range kvs {
			entry, err := ldf.Token{
				Key:   kv.Id,
				Type:  valueType,
				Value: kv.Value,
			}.Entry()
			if err != nil {
				return err
			}
			*entries = append(*entries, entry)
		}
		return nil
	}

	keys := struct {
		Strings  []KeyValue `xml:"string"`
		Utf8s    []KeyValue `xml:"utf8"`
		Int32s   []KeyValue `xml:"int32"`
		Uint32s  []KeyValue `xml:"uint32"`
		Int64s   []KeyValue `xml:"int64"`
		Uint64s  []KeyValue `xml:"uint64"`
		Float32s []KeyValue `xml:"float32"`
		Float64s []KeyValue `xml:"float64"`
		Bools    []KeyValue `xml:"bool"`
	}{}
	if err := d.DecodeElement(&keys, &start); err != nil {
		return err
	}

	var entries []ldf.Entry
	if err := appendEntries(&entries, keys.Strings, ldf.ValueTypeString); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Utf8s, ldf.ValueTypeUtf8); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Int32s, ldf.ValueTypeI32); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Uint32s, ldf.ValueTypeU32); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Int64s, ldf.ValueTypeI64); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Uint64s, ldf.ValueTypeU64); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Float32s, ldf.ValueTypeFloat); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Float64s, ldf.ValueTypeDouble); err != nil {
		return err
	}
	if err := appendEntries(&entries, keys.Bools, ldf.ValueTypeBool); err != nil {
		return err
	}
	*l = entries

	return nil
}

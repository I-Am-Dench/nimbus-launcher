package ldf

import (
	"bytes"
	"fmt"
	"reflect"
)

func valueToString(value reflect.Value) (string, error) {
	switch value.Kind() {
	case reflect.String:
		return value.String(), nil
	case reflect.Int, reflect.Int32:
		return fmt.Sprint(value.Int()), nil
	case reflect.Float32:
		return fmt.Sprint(float32(value.Float())), nil
	case reflect.Float64:
		return fmt.Sprint(value.Float()), nil
	case reflect.Uint, reflect.Uint32:
		return fmt.Sprint(value.Uint()), nil
	case reflect.Bool:
		if value.Bool() {
			return "1", nil
		} else {
			return "0", nil
		}
	default:
		return "", fmt.Errorf("cannot marshal type: %v", value.Type())
	}
}

func getTypeIndentifier(kind reflect.Kind) string {
	switch kind {
	case reflect.String:
		return "0"
	case reflect.Int, reflect.Int32:
		return "1"
	case reflect.Float32:
		return "3"
	case reflect.Float64:
		return "4"
	case reflect.Uint, reflect.Uint32:
		return "5"
	case reflect.Bool:
		return "7"
	default:
		panic(fmt.Errorf("cannot get type identifier for kind: %v", kind))
	}
}

func marshal(v any, delim string) ([]byte, error) {
	if v == nil {
		return []byte{}, nil
	}

	buffer := bytes.Buffer{}

	structValue := reflect.Indirect(reflect.ValueOf(v))
	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tagValue := field.Tag.Get("ldf")

		if tagValue == "-" {
			continue
		}

		key := tagValue
		if len(tagValue) == 0 {
			key = field.Name
		}

		fieldValue := structValue.Field(i)
		value, err := valueToString(fieldValue)
		if err != nil {
			return []byte{}, &MarshalError{err}
		}

		buffer.WriteString(key)
		buffer.WriteRune('=')
		buffer.WriteString(getTypeIndentifier(fieldValue.Kind()))
		buffer.WriteRune(':')
		buffer.WriteString(value)

		if i < structType.NumField()-1 {
			buffer.WriteString(delim)
		}
	}

	return buffer.Bytes(), nil
}

func Marshal(v any) ([]byte, error) {
	return marshal(v, ",")
}

func MarshalLines(v any) ([]byte, error) {
	return marshal(v, ",\n")
}

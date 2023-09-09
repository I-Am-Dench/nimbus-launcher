package luconfig

import (
	"bytes"
	"fmt"
	"reflect"
)

func Marshal(v any) ([]byte, error) {
	if v == nil {
		return []byte{}, nil
	}

	buffer := bytes.Buffer{}

	structValue := reflect.Indirect(reflect.ValueOf(v))
	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tagValue := field.Tag.Get("lucfg")

		if tagValue == "-" {
			continue
		}

		key := tagValue
		if len(tagValue) <= 0 {
			key = field.Name
		}

		fieldValue := structValue.Field(i)
		value, err := valueToString(fieldValue)
		if err != nil {
			return []byte{}, err
		}

		buffer.WriteString(key)
		buffer.WriteRune('=')
		buffer.WriteRune(typeToRune(fieldValue.Kind()))
		buffer.WriteRune(':')
		buffer.WriteString(value)

		if i < structType.NumField()-1 {
			buffer.WriteString(",\n")
		}
	}

	return buffer.Next(buffer.Len()), nil
}

func valueToString(value reflect.Value) (string, error) {
	switch value.Kind() {
	case reflect.String:
		return value.String(), nil
	case reflect.Int32, reflect.Int64:
		return fmt.Sprint(value.Int()), nil
	case reflect.Bool:
		if value.Bool() {
			return "1", nil
		} else {
			return "0", nil
		}
	default:
		return "", newMarshalError(fmt.Errorf("unusable value type '%s'", value.Type()))
	}
}

func typeToRune(kind reflect.Kind) rune {
	switch kind {
	case reflect.String:
		return '0'
	case reflect.Int32:
		return '1'
	case reflect.Int64:
		return '5'
	case reflect.Bool:
		return '7'
	default:
		return '0'
	}
}

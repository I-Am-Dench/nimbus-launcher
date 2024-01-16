package ldf

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type keyType int

const (
	String = keyType(iota)
	Signed32
	_
	Float
	Double
	Unsigned32
	_
	Boolean
)

type ldfValue struct {
	Kind  reflect.Kind
	Value string
}

func getKeyReflectKind(t keyType) reflect.Kind {
	switch t {
	case String:
		return reflect.String
	case Signed32:
		return reflect.Int32
	case Float:
		return reflect.Float32
	case Double:
		return reflect.Float64
	case Unsigned32:
		return reflect.Uint32
	case Boolean:
		return reflect.Bool
	default:
		return reflect.Invalid
	}
}

func parseValue(value string) (ldfValue, error) {
	typeIdentifier, rawValue, ok := strings.Cut(value, ":")
	if !ok {
		return ldfValue{}, fmt.Errorf("value `%s` is missing a type", value)
	}

	intIdentifier, err := strconv.Atoi(typeIdentifier)
	if err != nil {
		return ldfValue{}, fmt.Errorf("invalid type identifier: %v", err)
	}

	if intIdentifier < 0 {
		return ldfValue{}, fmt.Errorf("invalid type identifier: %d < 0", intIdentifier)
	}

	return ldfValue{
		Kind:  getKeyReflectKind(keyType(intIdentifier)),
		Value: strings.TrimSpace(rawValue),
	}, nil
}

func parseKeyValuePair(p string) (string, ldfValue, error) {
	key, typedValue, ok := strings.Cut(p, "=")
	if !ok {
		return "", ldfValue{}, fmt.Errorf("invalid key-value pair: %v", p)
	}

	value, err := parseValue(typedValue)
	if err != nil {
		return "", ldfValue{}, err
	}

	return strings.TrimSpace(key), value, nil
}

func Unmarshal(data []byte, v any) error {
	if v == nil || reflect.TypeOf(v).Kind() != reflect.Pointer {
		return nil
	}

	if reflect.ValueOf(v).IsNil() {
		return &UnmarshalError{errors.New("reference cannot be a nil pointer")}
	}

	keyValuePairs := make(map[string]ldfValue)

	s := string(data)
	rawPairs := strings.Split(s, ",")

	for _, p := range rawPairs {
		key, value, err := parseKeyValuePair(p)
		if err != nil {
			return err
		}

		keyValuePairs[key] = value
	}

	structValue := reflect.ValueOf(v).Elem()
	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tagValue := field.Tag.Get("ldf")

		if len(tagValue) <= 0 || tagValue == "-" {
			continue
		}

		ldfValue, ok := keyValuePairs[tagValue]
		if !ok {
			continue
		}

		value := structValue.Field(i)
		if !value.IsValid() || !value.CanSet() {
			continue
		}

		err := setStructField(value, ldfValue)
		if err != nil {
			return &UnmarshalError{err}
		}
	}

	return nil
}

func isCompatibleTypes(fieldKind, valueKind reflect.Kind) bool {
	switch fieldKind {
	case reflect.Int, reflect.Int32:
		return valueKind == reflect.Int32
	case reflect.Uint, reflect.Uint32:
		return valueKind == reflect.Uint32
	default:
		return fieldKind == valueKind
	}
}

func setStructField(field reflect.Value, ldfValue ldfValue) error {
	if !isCompatibleTypes(field.Kind(), ldfValue.Kind) {
		return fmt.Errorf("cannot set field type %v with %v", field.Type(), ldfValue.Kind)
	}

	switch ldfValue.Kind {
	case reflect.String:
		field.SetString(ldfValue.Value)
	case reflect.Int, reflect.Int32:
		i, err := strconv.ParseInt(ldfValue.Value, 10, 32)
		if err != nil {
			return err
		}

		field.SetInt(i)
	case reflect.Float32:
		f, err := strconv.ParseFloat(ldfValue.Value, 32)
		if err != nil {
			return err
		}

		field.SetFloat(f)
	case reflect.Float64:
		f, err := strconv.ParseFloat(ldfValue.Value, 64)
		if err != nil {
			return err
		}

		field.SetFloat(f)
	case reflect.Uint, reflect.Uint32:
		i, err := strconv.ParseUint(ldfValue.Value, 10, 32)
		if err != nil {
			return err
		}

		field.SetUint(i)
	case reflect.Bool:
		field.SetBool(ldfValue.Value == "1")
	default:
		return fmt.Errorf("cannot unmarshal LDF type: %v", ldfValue.Kind)
	}

	return nil
}

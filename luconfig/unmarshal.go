package luconfig

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type valueType int

const (
	String = valueType(iota)
	Integer
	Long
	Bool
)

type cfgValue struct {
	Type  valueType
	Value string
}

func runeToValueType(r rune) valueType {
	switch r {
	case '1':
		return Integer
	case '5':
		return Long
	case '7':
		return Bool
	case '0':
		fallthrough
	default:
		return String
	}
}

func getValueReflectKind(valueType valueType) reflect.Kind {
	switch valueType {
	case String:
		return reflect.String
	case Integer:
		return reflect.Int32
	case Long:
		return reflect.Int64
	case Bool:
		return reflect.Bool
	default:
		return reflect.Invalid
	}
}

func Unmarshal(data []byte, v any) error {
	if v == nil || reflect.TypeOf(v).Kind() != reflect.Pointer {
		return nil
	}

	if reflect.ValueOf(v).IsNil() {
		return newUnmarshalError("", fmt.Errorf("reference cannot be a nil pointer"))
	}

	configMapping := make(map[string]cfgValue)

	s := string(data)
	pairs := strings.Split(s, ",")

	for _, p := range pairs {
		key, value, err := parseKeyValuePair(p)
		if err != nil {
			return err
		}

		configMapping[key] = value
	}

	structValue := reflect.ValueOf(v).Elem()
	structType := structValue.Type()

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tagValue := field.Tag.Get("lucfg")

		if len(tagValue) <= 0 || tagValue == "-" {
			continue
		}

		cfgValue, ok := configMapping[tagValue]
		if !ok {
			continue
		}

		value := structValue.Field(i)
		if !value.IsValid() || !value.CanSet() {
			continue
		}

		err := setStructField(value, cfgValue)
		if err != nil {
			return newUnmarshalError("", err)
		}
	}

	return nil
}

// Sets field value from the config value based on its config type
func setStructField(value reflect.Value, cfgValue cfgValue) error {
	switch getValueReflectKind(cfgValue.Type) {
	case reflect.String:
		value.SetString(cfgValue.Value)
	case reflect.Int32:
		i, err := strconv.ParseInt(cfgValue.Value, 10, 32)
		if err != nil {
			return err
		}

		value.SetInt(i)
	case reflect.Int64:
		i, err := strconv.ParseInt(cfgValue.Value, 10, 64)
		if err != nil {
			return err
		}

		value.SetInt(i)
	case reflect.Bool:
		if cfgValue.Value == "1" {
			value.SetBool(true)
		} else {
			value.SetBool(false)
		}
	default:
		return fmt.Errorf("unusable reflect kind")
	}

	return nil
}

func parseKeyValuePair(p string) (string, cfgValue, error) {
	pair := strings.SplitN(p, "=", 2)
	if len(pair) < 2 {
		return "", cfgValue{}, newUnmarshalError(p, fmt.Errorf("invalid key-value pair"))
	}

	key, typedValue := pair[0], pair[1]

	value, err := parseValue(typedValue)
	if err != nil {
		return "", cfgValue{}, err
	}

	return strings.TrimSpace(key), value, nil
}

func parseValue(value string) (cfgValue, error) {
	pair := strings.SplitN(value, ":", 2)
	if len(pair) < 1 {
		return cfgValue{}, newUnmarshalError(value, fmt.Errorf("invalid value"))
	}

	if len(pair) < 2 {
		return cfgValue{}, newUnmarshalError(value, fmt.Errorf("missing value type"))
	}

	typeIdentifier, rawValue := pair[0], strings.TrimSpace(pair[1])
	if len(typeIdentifier) != 1 {
		return cfgValue{}, newUnmarshalError(value, fmt.Errorf("invalid type identifier '%s'", typeIdentifier))
	}

	valueType := runeToValueType(rune(typeIdentifier[0]))
	if !isValidValue(rawValue, valueType) {
		return cfgValue{}, newUnmarshalError(value, fmt.Errorf("incompatible types"))
	}

	return cfgValue{
		Type:  valueType,
		Value: rawValue,
	}, nil
}

// Soft type check for whether or not the raw string can be converted to its struct type
func isValidValue(value string, valueType valueType) bool {
	switch valueType {
	case String:
		return true
	case Integer, Long:
		for _, r := range value {
			if !(r == '.' || r == '-' || ('0' <= r && r <= '9')) {
				return false
			}
		}
		return true
	case Bool:
		return len(value) == 1 && (value == "1" || value == "0")
	default:
		return false
	}
}

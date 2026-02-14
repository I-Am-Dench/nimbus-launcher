// Custom persistent preferences package
// since fyne.Preferences is very limiting.
package configfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"fyne.io/fyne/v2/data/binding"
)

type ConfigFile[T any] struct {
	path         string
	binding      binding.Item[T]
	defaultValue func() T
}

func (c *ConfigFile[T]) Binding() binding.Item[T] {
	return c.binding
}

func (c *ConfigFile[T]) Get() T {
	value, _ := c.binding.Get()
	return value
}

func (c *ConfigFile[T]) Load() (value T, err error) {
	data, err := os.ReadFile(c.path)
	if errors.Is(err, os.ErrNotExist) {
		value = c.defaultValue()
		if err := c.Save(value); err != nil {
			return value, err
		}
		c.binding.Set(value)
		return value, nil
	}

	if err != nil {
		return value, fmt.Errorf("load config file: %v", err)
	}

	if err := json.Unmarshal(data, &value); err != nil {
		return value, fmt.Errorf("load config file: %v", err)
	}

	c.binding.Set(value)
	return value, nil
}

func (c *ConfigFile[T]) Save(value T) error {
	c.binding.Set(value)

	data, err := json.MarshalIndent(value, "", "    ")
	if err != nil {
		return fmt.Errorf("save config file: %v", err)
	}

	if err := os.WriteFile(c.path, data, 0755); err != nil {
		return fmt.Errorf("save config file: %v", err)
	}
	return nil
}

func New[T any](path string, defaultValue func() T) *ConfigFile[T] {
	return &ConfigFile[T]{
		path:         path,
		binding:      binding.NewItem(func(_, _ T) bool { return false }),
		defaultValue: defaultValue,
	}
}

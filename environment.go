package main

import (
	"fmt"
	"reflect"
	"strings"
)

// Environment represents a single environment. Since it can have a
// variable depth, this provides some helper functions for accessing
// information.
type Environment map[interface{}]interface{}

// Flatten the environment variables where hierarchy uses dot-notation
// instead of nested maps. Key/Value pairs are added to the given
// map. Interpolation is also done on the environment values from the
// given responses.
func (e Environment) Flatten(responses map[string]string) {
	result := map[string]string{}
	flattenHelperII(e, result, "environment")

	// Do interpolation for the environment variables.
	for key, value := range result {
		responses[key] = interpolate(value, responses)
	}
}

func flattenHelperII(e map[interface{}]interface{}, result map[string]string, prefix string) {
	for k, v := range e {
		key, ok := k.(string)
		if !ok {
			return
		}

		t := reflect.TypeOf(v).Kind()
		if t == reflect.Int || t == reflect.Float32 || t == reflect.Float64 || t == reflect.String || t == reflect.Bool {
			result[prefix+"."+key] = fmt.Sprintf("%v", v)
		} else if i, ok := v.(map[interface{}]interface{}); ok {
			flattenHelperII(i, result, prefix+"."+key)
		} else if s, ok := v.(map[string]interface{}); ok {
			flattenHelperSI(s, result, prefix+"."+key)
		}
	}
}

func flattenHelperSI(e map[string]interface{}, result map[string]string, prefix string) {
	for key, v := range e {
		t := reflect.TypeOf(v).Kind()
		if t == reflect.Int || t == reflect.Float32 || t == reflect.Float64 || t == reflect.String || t == reflect.Bool {
			result[prefix+"."+key] = fmt.Sprintf("%v", v)
		} else if i, ok := v.(map[interface{}]interface{}); ok {
			flattenHelperII(i, result, prefix+"."+key)
		} else if s, ok := v.(map[string]interface{}); ok {
			flattenHelperSI(s, result, prefix+"."+key)
		}
	}
}

// Get the value of the given key. Keys of a depth of more than one
// should be separated by a dot (.). For example, 'auth.token' would
// get the values of token in the auth map.
func (e Environment) Get(key string) (string, error) {
	parts := strings.Split(key, ".")
	i, err := findii(e, parts)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v", i), nil
}

func findii(e map[interface{}]interface{}, parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return "", ErrKeyNotFound
	}

	v := e[parts[0]]

	// We are at the end, expect a non-recursive type.
	if len(parts) == 1 {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Int, reflect.Float32, reflect.Float64, reflect.String, reflect.Bool:
			return v, nil
		default:
			return nil, ErrUnsupportedType
		}
	}

	// We can get a couple different types that are recursive, so check them.
	if i, ok := v.(map[interface{}]interface{}); ok {
		return findii(i, parts[1:])
	} else if s, ok := v.(map[string]interface{}); ok {
		return findsi(s, parts[1:])
	}

	return nil, ErrKeyNotFound
}

func findsi(e map[string]interface{}, parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return "", ErrKeyNotFound
	}

	v := e[parts[0]]

	// We are at the end, expect a non-recursive type.
	if len(parts) == 1 {
		switch reflect.TypeOf(v).Kind() {
		case reflect.Int, reflect.Float32, reflect.Float64, reflect.String, reflect.Bool:
			return v, nil
		default:
			return nil, ErrUnsupportedType
		}
	}

	// We can get a couple different types that are recursive, so check them.
	if i, ok := v.(map[interface{}]interface{}); ok {
		return findii(i, parts[1:])
	} else if s, ok := v.(map[string]interface{}); ok {
		return findsi(s, parts[1:])
	}

	return nil, ErrKeyNotFound
}

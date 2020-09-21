package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

type Response struct {
	When        time.Time         `yaml:"when"`
	Status      string            `yaml:"status"`
	StatusCode  int               `yaml:"status-code"`
	Duration    time.Duration     `yaml:"duration"`
	Cookies     map[string]string `yaml:"cookies"`
	Headers     map[string]string `yaml:"headers"`
	Body        string            `yaml:"body"`
	RawRequest  string            `yaml:"raw-request"`
	RawResponse string            `yaml:"raw-response"`
}

// Flatten the JSON of the body to the given map where
// hierarchy uses dot-notation instead of nested maps.
func (r *Response) Flatten(m map[string]string, name string) error {
	e := map[string]interface{}{}
	err := json.Unmarshal([]byte(r.Body), &e)
	if err != nil {
		return err
	}

	flattenHelperJSON(e, m, "responses."+name)
	return nil
}

func flattenHelperJSON(e map[string]interface{}, result map[string]string, prefix string) {
	for key, v := range e {
		t := reflect.TypeOf(v).Kind()
		if t == reflect.Int || t == reflect.Float32 || t == reflect.Float64 || t == reflect.String || t == reflect.Bool {
			result[prefix+"."+key] = fmt.Sprintf("%v", v)
		} else if s, ok := v.(map[string]interface{}); ok {
			flattenHelperJSON(s, result, prefix+"."+key)
		}
	}
}

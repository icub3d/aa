package main

import (
	"regexp"
	"strings"
)

type Request struct {
	Description    string            `yaml:"description,omitempty"`
	URL            string            `yaml:"url"`
	Method         string            `yaml:"method"`
	Headers        map[string]string `yaml:"headers"`
	Authentication map[string]string `yaml:"authentication"`
	Query          map[string]string `yaml:"query"`
	Body           Body              `yaml:"body,omitempty"`
}

type Body struct {
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

func (r *Request) Interpolate(vars map[string]string) {
	r.URL = interpolate(r.URL, vars)
	r.Method = interpolate(r.Method, vars)
	r.Body.Type = interpolate(r.Body.Type, vars)
	r.Body.Value = interpolate(r.Body.Value, vars)

	for k, v := range r.Headers {
		r.Headers[k] = interpolate(v, vars)
	}

	for k, v := range r.Authentication {
		r.Authentication[k] = interpolate(v, vars)
	}

	for k, v := range r.Query {
		r.Query[k] = interpolate(v, vars)
	}
}

var re = regexp.MustCompile(`\{\{[^\}]*\}\}`)

func interpolate(s string, vars map[string]string) string {
	matches := re.FindAllString(s, -1)
	for _, match := range matches {
		m := strings.Trim(match, "{}")
		if v, ok := vars[m]; ok {
			s = strings.ReplaceAll(s, match, v)
		}
	}
	return s
}

package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/gookit/color"
	"gopkg.in/yaml.v3"
)

func createRequestBody(req *http.Request, body Body) (io.ReadCloser, error) {
	switch body.Type {
	case "raw":
		return handleRawBodyRequest(body.Value)
	case "file":
		return handleFileBodyRequest(body.Value)
	case "multipart":
		return handleMultipartBodyRequest(req, body.Value)
	case "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unexpected body type: %v", body.Type)
	}
}

func handleRawBodyRequest(s string) (io.ReadCloser, error) {
	color.Blue.Printf("%s\n", s)
	return ioutil.NopCloser(bytes.NewBufferString(s)), nil

}

func handleFileBodyRequest(s string) (io.ReadCloser, error) {
	body, err := os.Open(s)
	if err != nil {
		return nil, fmt.Errorf("opening file '%v': %v", s, err)
	}
	color.Blue.Printf("<file contents of '%s'>\n", s)
	return body, nil
}

type MultiPartPart struct {
	Type  string `yaml:"type"`
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

func handleMultipartBodyRequest(req *http.Request, s string) (io.ReadCloser, error) {
	// Parse the yaml
	parts := []MultiPartPart{}
	err := yaml.Unmarshal([]byte(s), &parts)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	req.Header.Add("Content-Type", mw.FormDataContentType())
	color.Blue.Printf("Content-Type: %s\n", mw.FormDataContentType())

	go func() {
		defer pw.Close()
		defer mw.Close()
		for _, part := range parts {
			var r io.ReadCloser
			var w io.Writer
			var err error

			color.Blue.Printf("--- part: %v ---\n", part.Name)
			switch part.Type {
			case "raw":
				r, err = handleRawBodyRequest(part.Value)
				if err == nil {
					w, err = mw.CreateFormField(part.Name)
				}
			case "file":
				r, err = handleFileBodyRequest(part.Value)
				if err == nil {
					w, err = mw.CreateFormFile(part.Name, part.Value)
				}
			default:
				err = fmt.Errorf("unsupported part type: %s", part.Type)
			}
			if err != nil {
				color.Red.Printf("writing body part '%v': %v", part.Name, err)
				return
			}
			_, err = io.Copy(w, r)
			r.Close()
			if err != nil {
				color.Red.Printf("writing body part '%v': %v", part.Name, err)
				return
			}
		}
	}()

	return pr, nil
}

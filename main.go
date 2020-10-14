package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

func main() {
	app := &cli.App{
		Name:     "aa",
		Usage:    "API Automation for the command line!",
		Version:  "v0.1.3",
		Compiled: time.Now(),
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "Joshua Marsh",
				Email: "joshua@themarshians.com",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"cfg", "c"},
				EnvVars: []string{"AA_CONFIG", "AA_CFG"},
				Value:   "aa",
				Usage:   "the file or folder containting environments, requests, and responses to use",
			},
			&cli.StringFlag{
				Name:     "environment",
				Aliases:  []string{"env", "e"},
				EnvVars:  []string{"AA_ENVIRONMENT", "AA_ENV"},
				Usage:    "the name of the environment to use for string interpolation",
				Required: true,
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "requests",
				Aliases: []string{"req", "r"},
				Subcommands: []*cli.Command{
					{
						Name:    "run",
						Aliases: []string{"r"},
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:    "json",
								Aliases: []string{"j"},
								EnvVars: []string{"AA_RUN_JSON"},
								Usage:   "pretty print json responses",
							},
							&cli.StringFlag{
								Name:    "body",
								Aliases: []string{"b"},
								EnvVars: []string{"AA_RUN_BODY"},
								Usage:   "save response body to given file instead of printing it",
							},
						},
						Usage:  "run a list of requests",
						Action: wrap(requestrun),
					},
					{
						Name:    "list",
						Aliases: []string{"l", "ls"},
						Usage:   "list requests",
						Action:  wrap(requestlist),
					},
				},
			},
		},
	}
	app.Run(os.Args)
}

func wrap(f func(ctx *cli.Context, cfg *Config, env Environment) error) cli.ActionFunc {
	return func(c *cli.Context) error {
		// Get our confign
		cfg, err := NewConfig(c.String("config"))
		if err != nil {
			return cli.Exit(color.Red.Sprintf("loading config (%v): %v\n", c.String("config"), err), -1)
		}

		// Make sure we have an environment.
		env, ok := cfg.Environments[c.String("environment")]
		if !ok {
			return cli.Exit(color.Red.Sprintf("environment '%v' not found\n", c.String("environment")), -1)
		}

		return f(c, cfg, env)
	}
}

func requestlist(c *cli.Context, cfg *Config, env Environment) error {
	for r, v := range cfg.Requests {
		color.Magenta.Printf("%v", r)
		if v.Description != "" {
			fmt.Print(" - ")
			color.Green.Printf("%v", v.Description)
		}
		fmt.Print("\n")
	}
	return nil
}

func requestrun(c *cli.Context, cfg *Config, env Environment) error {
	if !c.Args().Present() {
		return cli.Exit(color.Red.Sprintf("run expects at least one request name"), -1)
	}

	// Run for each request.
	for x := 0; x < c.Args().Len(); x++ {

		// Flatten the interpolation data.
		vars := map[string]string{}
		for k, v := range cfg.Responses {
			v.Flatten(vars, k)
		}
		env.Flatten(vars)

		// Print out the name.
		name := c.Args().Get(x)
		color.Magenta.Println("================================================================")
		color.Magenta.Println(name)
		color.Magenta.Println("================================================================")
		req, ok := cfg.Requests[name]
		if !ok {
			return cli.Exit(color.Red.Sprintf("request '%v' not found", name), -1)
		}

		req.Interpolate(vars)

		resp, err := run(c, name, req, cfg.Preferences)
		if err != nil {
			return cli.Exit(color.Red.Sprintf("running %v: %v", name, err), -1)
		}

		// Flatten for upcoming runs.
		cfg.Responses[name] = *resp

		// Also save to disk for future executions.
		y, err := yaml.Marshal(&Config{
			Responses: map[string]Response{
				name: *resp,
			},
		})
		if err != nil {
			return cli.Exit(color.Red.Sprintf("marshalling response yaml: %v", err), -1)
		}
		err = ioutil.WriteFile(filepath.Join(c.String("config"), name+"-response.yaml"), y, 0660)
		if err != nil {
			return cli.Exit(color.Red.Sprintf("saving response yaml: %v", err), -1)
		}
	}
	return nil
}

func run(ctx *cli.Context, name string, r Request, prefs map[string]string) (*Response, error) {
	in, err := os.Create(filepath.Join(ctx.String("config"), name+"-response.raw"))
	if err != nil {
		return nil, fmt.Errorf("creating response raw file: %v", err)
	}
	defer in.Close()
	out, err := os.Create(filepath.Join(ctx.String("config"), name+"-request.raw"))
	if err != nil {
		return nil, fmt.Errorf("creating request raw file: %v", err)
	}
	defer out.Close()

	// Create our client
	tlsCfg := &tls.Config{}
	if ignore, ok := prefs["ignore-certs"]; ok && ignore == "true" {
		tlsCfg.InsecureSkipVerify = true
	}

	client := http.Client{}
	client.Transport = NewHelperTransport(in, out, tlsCfg)

	req := &http.Request{
		Method: r.Method,
	}

	// Setup the URL
	u, err := url.Parse(r.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing url: %v", err)
	}
	q := u.Query()
	for k, v := range r.Query {
		q.Add(k, v)
	}
	u.RawQuery = q.Encode()
	req.URL = u
	color.Blue.Printf("%v %v\n", req.Method, req.URL)

	// Create Headers
	req.Header = http.Header{}
	for k, v := range r.Headers {
		req.Header.Add(k, v)
		color.Blue.Printf("%v: %v\n", k, v)
	}

	// Setup Authentication
	if authType, ok := r.Authentication["type"]; ok {
		switch strings.ToLower(authType) {
		case "bearer":
			req.Header.Add("Authorization", "Bearer "+r.Authentication["token"])
			color.Blue.Printf("%v: %v\n", "Authorization", "Bearer "+r.Authentication["token"])
		}
	}

	// Setup the body
	body, err := createRequestBody(req, r.Body)
	if err != nil {
		return nil, fmt.Errorf("creating request body: %v", err)
	}
	req.Body = body
	color.Blue.Printf("\n")

	// Do request
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %v", err)
	}

	color.Green.Printf("%v %v\n", resp.Proto, resp.Status)
	for k, v := range resp.Header {
		color.Green.Printf("%v: %v\n", k, v)
	}
	color.Green.Printf("\n")

	// Copy response body
	ww := []io.Writer{}
	b := &bytes.Buffer{}
	ww = append(ww, b)

	// Add a file writer.
	if f := ctx.String("body"); f != "" {
		b, err := os.Create(f)
		if err != nil {
			return nil, fmt.Errorf("opening body file '%v': %v", f, err)
		}
		defer b.Close()
		ww = append(ww, b)
		color.Green.Printf("<body saved to '%v'>\n", f)
	}

	_, err = io.Copy(io.MultiWriter(ww...), resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading body: %v", err)
	}
	resp.Body.Close()

	if ctx.Bool("json") && strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		tmp := map[string]interface{}{}
		if err := json.Unmarshal(b.Bytes(), &tmp); err == nil {
			buf, err := json.MarshalIndent(tmp, "", "     ")
			if err == nil {
				b = bytes.NewBuffer(buf)
			}
		}
	}

	if ctx.String("body") == "" {
		color.Green.Printf("%v\n", b.String())
	}

	duration := time.Since(start)
	color.Magenta.Printf("\nduration: %v\n", duration)

	// Create and return response information.
	response := &Response{
		When:       time.Now(),
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Duration:   duration,
		Body:       b.String(),
	}

	response.Headers = map[string]string{}
	for k, v := range resp.Header {
		response.Headers[k] = fmt.Sprintf("%s", v)
	}

	return response, nil
}

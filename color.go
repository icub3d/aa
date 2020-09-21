package main

import "github.com/gookit/color"

type ColorPrinter struct {
	c color.Color
}

func (c *ColorPrinter) Write(p []byte) (int, error) {
	c.c.Printf("%s", string(p))
	return len(p), nil
}

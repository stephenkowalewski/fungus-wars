//go:build ignore
// +build ignore

// This program generates static/modal/how_to_play.html
// Run it via `go generate`.

package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

func main() {
	files := []string{"how_to_play.md"}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)

	for _, f := range files {
		in, err := os.ReadFile(f)
		if err != nil {
			log.Fatal(err)
		}

		var out strings.Builder
		if err := md.Convert(in, &out); err != nil {
			log.Fatal(err)
		}

		htmlName := "static" + string(os.PathSeparator) + "modal" + string(os.PathSeparator) + strings.TrimSuffix(f, ".md") + ".html"
		if err := ioutil.WriteFile(htmlName, []byte(out.String()), 0644); err != nil {
			log.Fatal(err)
		}

		log.Println("Wrote:", htmlName)
	}
}

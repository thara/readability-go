package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	readability "github.com/thara/readability-go"
)

func main() {
	baseURL := flag.String("base-url", "", "Base URL for resolving relative links")
	jsonOutput := flag.Bool("json", false, "Output as JSON")
	check := flag.Bool("check", false, "Only check if document is probably readable")
	flag.Parse()

	var r io.Reader
	var documentURI string

	arg := flag.Arg(0)
	switch {
	case arg == "" || arg == "-":
		r = os.Stdin
		documentURI = "http://localhost/"
		if *baseURL != "" {
			documentURI = *baseURL
		}
	case strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://"):
		resp, err := http.Get(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching URL: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		r = resp.Body
		documentURI = arg
		if *baseURL != "" {
			documentURI = *baseURL
		}
	default:
		f, err := os.Open(arg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		r = f
		documentURI = "file://" + arg
		if *baseURL != "" {
			documentURI = *baseURL
		}
	}

	if *check {
		data, err := io.ReadAll(r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			os.Exit(1)
		}
		readable, err := readability.IsProbablyReaderable(strings.NewReader(string(data)))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if readable {
			fmt.Println("true")
		} else {
			fmt.Println("false")
			os.Exit(1)
		}
		return
	}

	article, err := readability.Parse(r, documentURI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if article == nil {
		fmt.Fprintln(os.Stderr, "No article content found")
		os.Exit(1)
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		if err := enc.Encode(article); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println(article.Content)
	}
}

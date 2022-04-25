package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/spf13/pflag"
)

const apiEndpoint string = "api.github.com"
const apiPath string = "gitignore/templates"
const outFile string = ".gitignore"

type Mode int

const (
	List Mode = 0
	Get  Mode = 1
)

type Args struct {
	mode        Mode
	pathSpec    string
	ignoreSpecs []string
}

// Makes API request with the specified args and returns the buffer.
func invokeRequest(args *Args) []byte {
	if args.mode == List {
		url := url.URL{
			Scheme: "https",
			Host:   apiEndpoint,
			Path:   apiPath,
		}

		resp, err := http.Get(url.String())
		bailif(err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		bailif(err)

		return body
	} else {
		fmt.Printf("getting templates: %v\n", args.ignoreSpecs)

		var buffer bytes.Buffer
		var client http.Client

		req := http.Request{
			Method: "GET",
			Header: http.Header{
				"Accept": {"application/vnd.github.v3.raw"},
			},
			URL: &url.URL{
				Scheme: "https",
				Host:   apiEndpoint,
			},
		}

		for _, spec := range args.ignoreSpecs {
			req.URL.Path = path.Join("/", apiPath, spec)

			resp, err := client.Do(&req)
			bailif(err)
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				fmt.Printf("oops, got a %d, skipping! check your spelling: %s\n",
					resp.StatusCode, spec)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			bailif(err)

			_, err = buffer.WriteString(fmt.Sprintf("# # # %s # # #\n", spec))
			bailif(err)

			_, err = buffer.Write(body)
			bailif(err)
		}
		return buffer.Bytes()
	}
}

// Removes the fluff and pretty prints available templates.
func prettyPrint(buffer []byte) {
	avail := string(buffer)
	avail = strings.ReplaceAll((strings.ReplaceAll(
		strings.TrimSuffix(strings.TrimPrefix(avail, "["), "]"),
		"\"", "")),
		",", "\n")
	fmt.Printf("%s\n", avail)
}

// Writes buffer to file.
func writeBuffer(buffer []byte, args *Args) {
	path := path.Join(args.pathSpec, outFile)
	bailif(os.WriteFile(path, buffer, 0644))
	fmt.Printf("new file: %s\n", path)
}

// Parses input arguments.
// The following command forms are permitted:
//	ignore help
//	ignore list
//	ignore get [spec[?...]]
//	ignore get [spec[?...]] -path=[pathspec]
func parseArgs() Args {
	args := Args{}

	flags := pflag.NewFlagSet(os.Args[0], pflag.PanicOnError)
	flags.Usage = func() {
		fmt.Fprint(os.Stderr, help())
		os.Exit(0)
	}

	if len(os.Args) < 2 {
		flags.Usage()
	}

	switch os.Args[1] {
	case "list":
		args.mode = List
	case "get":
		flags.StringVar(&args.pathSpec, "path", cwd(), "Specify the output directory")
		bailif(flags.Parse(os.Args[2:]))
		args.mode = Get
		args.ignoreSpecs = flags.Args()
	case "help":
		flags.Usage()
	default:
		panic(fmt.Errorf("unknown command"))
	}

	return args
}

// Panics on error.
func bailif(err any) {
	if err != nil {
		panic(err)
	}
}

// Returns the current working directory.
func cwd() string {
	wd, err := os.Getwd()
	bailif(err)
	return wd
}

// Returns the help string
func help() string {
	return `Gets .gitignore templates for you <3

Usage:
    ignore <command> [arguments]

Commands: 
    list                list all available templates
    get [spec [?...]]   get a template
    help                print help

Arguments:
  -path string
        Specify the output directory (default ".")

Examples:
    ignore list
    ignore get C Julia TeX
    ignore get VisualStudio
`
}

func main() {
	args := parseArgs()
	bytes := invokeRequest(&args)
	if args.mode == List {
		prettyPrint(bytes)
	} else {
		writeBuffer(bytes, &args)
	}
}

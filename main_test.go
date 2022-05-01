package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestList(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `["FirstTemplate","SecondTemplate"]`)
		}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true}

	args := Args{mode: List}
	reqEndpoint := strings.TrimPrefix(srv.URL, "https://")
	reqPath := apiPath

	resp := invokeRequest(&args, reqEndpoint, reqPath)

	stdout, _ := captureStdout(func() any {
		prettyPrint(resp)
		return nil
	})

	expectedStdout := "FirstTemplate\nSecondTemplate\n"

	if expectedStdout != stdout {
		t.Fatalf("invalid list cmd output! expected: %q, got: %q",
			expectedStdout, stdout)
	}
}

func TestGet(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "# # # FirstTemplate # # #\n**/.first_dir\n")
		}))
	srv.EnableHTTP2 = true
	srv.TLS = &tls.Config{InsecureSkipVerify: true}
	srv.StartTLS()
	defer srv.Close()

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true}

	args := Args{
		mode:        Get,
		ignoreSpecs: []string{"FirstTemplate"},
		pathSpec:    cwd(),
	}
	reqEndpoint := strings.TrimPrefix(srv.URL, "https://")
	reqPath := apiPath

	stdout, got := captureStdout(func() any {
		return invokeRequest(&args, reqEndpoint, reqPath)
	})

	expected := []byte("# # # FirstTemplate # # #\n**/.first_dir\n")

	if bytes.Equal(expected, got.([]byte)) {
		t.Fatalf("invalid get cmd buffer! expected: %q, got: %q", expected, got)
	}

	expectedStdout := fmt.Sprintf("getting templates: %v\n",
		[]string{"FirstTemplate"})

	if expectedStdout != stdout {
		t.Fatalf("invalid get cmd output! expected: %q, got: %q",
			expectedStdout, stdout)
	}
}

func TestParseArgs(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"ignore", "list"}
	parsed := parseArgs()

	if parsed.mode != List {
		t.Fatalf("invalid parsed args! got: %v, for input: %v", parsed, os.Args)
	}

	os.Args = []string{"ignore", "get", "C", "Julia"}
	parsed = parseArgs()

	if parsed.mode != Get ||
		len(parsed.ignoreSpecs) != len([]string{"C", "Julia"}) {
		t.Fatalf("invalid parsed args! got: %v, for input: %v", parsed, os.Args)
	}

	os.Args = []string{"ignore", "get", "C", "--path=.."}
	parsed = parseArgs()

	if parsed.mode != Get ||
		len(parsed.ignoreSpecs) != len([]string{"C"}) ||
		parsed.pathSpec != ".." {
		t.Fatalf("invalid parsed args! got: %v, for input: %v", parsed, os.Args)
	}
}

func captureStdout(f func() any) (string, any) {
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	r, w, _ := os.Pipe()
	os.Stdout = w

	retVal := f()

	outChan := make(chan string)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outChan <- buf.String()
	}()

	w.Close()

	return <-outChan, retVal
}

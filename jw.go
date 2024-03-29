package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
)

// A jenkinsReader exposes the streaming console output for a given jenkins job
// as an io.Reader.
type jenkinsReader struct {
	username string
	key      string
	base     *url.URL
	client   http.Client
	response *http.Response
	position string
}

// Creates the jenkinsReader. Returns an error if something is wrong with the URL.
func newJenkinsReader(username, key, job string) (reader jenkinsReader, err error) {
	transport := http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: insecure,
	}

	reader = jenkinsReader{
		username: username,
		key:      key,
		position: "0",
		client: http.Client{
			Transport: transport,
		},
	}
	reader.base, err = url.Parse(job)
	if err != nil {
		return
	}

	// if someone just copy+pasted the URL from the browser, strip off the console part
	if dir, file := path.Split(reader.base.Path); file == "console" {
		reader.base.Path = dir
	}

	reader.base.Path = path.Join(reader.base.Path, "/logText/progressiveText")
	return
}

func (reader *jenkinsReader) setAuth(req *http.Request) {
	req.SetBasicAuth(reader.username, reader.key)
}

// Performs a HEAD request to make sure the URL
// and credentials are correct.
func (reader *jenkinsReader) check() (err error) {
	req, err := http.NewRequest("HEAD", reader.base.String(), nil)
	if err != nil {
		return
	}
	reader.setAuth(req)
	res, err := reader.client.Do(req)
	if err != nil {
		return
	}

	_, err = io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		return
	}
	err = res.Body.Close()
	if err != nil {
		return
	}

	if !strings.HasPrefix(res.Status, "2") {
		err = errors.New("bad response for head: " + res.Status)
	}

	return
}

// requests a chunk of console output at the given offset.
func (reader *jenkinsReader) getLogAt(offset string) (err error) {
	url := *reader.base
	q := url.Query()
	q.Set("start", offset)
	url.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return
	}

	reader.setAuth(req)
	reader.response, err = reader.client.Do(req)

	return
}

// Reads bytes from the current chunk of console output.
// If the end of the current chunk is reached, the current position
// in the console output is saved, and the X-More-Data header is
// checked to see if there's more. If there isn't, EOF is reported.
// TODO: how aggressively will this be polled?
// TODO: should we care about timeouts?
func (reader *jenkinsReader) Read(p []byte) (n int, err error) {
	if reader.response == nil {
		err = reader.getLogAt(reader.position)
		if err != nil {
			return
		}
	}

	n, err = reader.response.Body.Read(p)

	if err == io.EOF {
		err = reader.response.Body.Close()
		if err != nil {
			return
		}
		reader.position = reader.response.Header.Get("X-Text-Size")
		if reader.response.Header.Get("X-More-Data") != "true" {
			err = io.EOF
		}
		reader.response = nil
	}

	return
}

var (
	insecure bool
	user     string
	key      string
	jobURL   string
	allGood  bool
)

func init() {
	flag.BoolVar(&insecure, "k", false, "Does not check TLS certs when set.")

	user = os.Getenv("JENKINS_USER")
	key = os.Getenv("JENKINS_KEY")

	allGood = true

	if user == "" {
		fmt.Fprintln(os.Stderr, "JENKINS_USER must be set")
		allGood = false
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "JENKINS_KEY must be set")
		allGood = false
	}

	flag.Parse()

	jobURL = flag.Arg(0)
	if jobURL == "" {
		fmt.Fprintln(os.Stderr, "url for jenkins job must be given")
		allGood = false
	}

	if !allGood {
		os.Exit(1)
	}
}

// Print to stderr and exit with a status of 1.
func die(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(1)
}

func main() {
	reader, err := newJenkinsReader(user, key, jobURL)
	if err != nil {
		die(err)
	}

	err = reader.check()
	if err != nil {
		die(err)
	}

	_, err = io.Copy(os.Stdout, &reader)

	if err != nil {
		die(err)
	}
}

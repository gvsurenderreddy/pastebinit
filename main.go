package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
)

var (
	baseuri string

	username = os.Getenv("PASTEBINIT_USERNAME")
	password = os.Getenv("PASTEBINIT_PASS")
)

// readFromStdin returns everything in stdin.
func readFromStdin() []byte {
	stdin, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		logrus.Fatalf("reading from stdin failed: %v", err)
	}
	return stdin
}

// readFromFile returns the contents of a file.
func readFromFile(filename string) []byte {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		logrus.Fatalf("No such file or directory: %q", filename)
	}

	file, err := ioutil.ReadFile(filename)
	if err != nil {
		logrus.Fatalf("reading from file %q failed: %v", filename, err)
	}
	return file
}

// postPaste uploads the paste content to the server
// and returns the paste URI.
func postPaste(content []byte) (string, error) {
	// create the request
	req, err := http.NewRequest("POST", baseuri+"paste", bytes.NewBuffer(content))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(username, password)

	// do the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request to %spaste failed: %v", baseuri, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return "", fmt.Errorf("Unauthorized. Please check your username and pass. %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body failed: %v", err)
	}

	var response map[string]string
	if err = json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("parsing body as json failed: %v", err)
	}

	if respError, ok := response["error"]; ok {
		return "", fmt.Errorf("server responded with %s", respError)
	}

	pasteURI, ok := response["uri"]
	if !ok {
		return "", fmt.Errorf("what the hell did we get back even? %s", string(body))
	}

	return pasteURI, nil
}

func init() {
	flag.StringVar(&baseuri, "b", "https://paste.j3ss.co/", "pastebin base url")
	flag.Parse()

	// make sure uri ends with trailing /
	if !strings.HasSuffix(baseuri, "/") {
		baseuri += "/"
	}
	// make sure it starts with http(s)://
	if !strings.HasPrefix(baseuri, "http") {
		baseuri = "http://" + baseuri
	}

	// make sure we have a username and password
	if username == "" || password == "" {
		logrus.Fatalf("you need to pass the PASTEBINIT_USERNAME and PASTEBINIT_PASS env variables")
	}
}

func main() {
	args := flag.Args()

	// check if we are reading from a file or stdin
	var content []byte
	if len(args) == 0 {
		content = readFromStdin()
	} else {
		filename := args[0]
		content = readFromFile(filename)
	}

	pasteURI, err := postPaste(content)
	if err != nil {
		logrus.Fatal(err)
	}

	fmt.Printf("Your paste has been uploaded here:\n%s\nthe raw object is here: %s/raw", pasteURI, pasteURI)
}

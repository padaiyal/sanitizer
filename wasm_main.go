package main

//goland:noinspection
import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall/js"
)

type Config struct {
	SupportedFileExtensions []string `json:"SupportedFileExtensions"`
	SupportedActions        []string `json:"SupportedActions"`
}

type RuleSet struct {
	Description string `yaml:"description"`
	Format      string `yaml:"format"`
	Rules       map[string]struct {
		Description string `yaml:"description"`
		Action      string `yaml:"action"`
	} `yaml:"rules"`
}

var config = Config{}
var ruleSets = map[string]RuleSet{}
var document = js.Global().Get("document")

func errorFollowUp(err error, exit bool) {
	println("error=", err.Error())
	if exit {
		os.Exit(1)
	}
}

func getResponse[T interface{}](url string, responseStruct *T) ([]byte, error) {
	/**
	Make an HTTP GET request to the specified URL and marshal the response body into
	the provided struct object.
	*/
	response, err := http.Get(url)
	if response == nil || response.StatusCode != 200 || err != nil {
		return nil, err
	}
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	response.Body.Close()
	if responseStruct != nil {
		contentType := response.Header.Get("Content-Type")
		fmt.Println("Content-Type = ", contentType)
		if contentType == "application/json" {
			err = json.Unmarshal(bodyBytes, responseStruct)
		} else if contentType == "application/yaml" || contentType == "text/yaml" {
			err = yaml.Unmarshal(bodyBytes, responseStruct)
		}
	}
	return bodyBytes, err
}

func getRuleFilePath(fileExtension string) string {
	return "rules/" + fileExtension + ".yaml"
}

func toPrettyJson(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")
	return out.Bytes(), err
}

func generateSanitizedFileName(filePath string) string {
	splitIndex := strings.LastIndex(filePath, ".")
	return filePath[:splitIndex] + "_sanitized." + filePath[splitIndex+1:]
}

func sanitizeCallback(_ js.Value, _ []js.Value) any {
	/**
	Callback when an input file is selected to be sanitized.
	*/
	uploadButton := document.Call("getElementById", "upload_button")
	file := uploadButton.Get("files").Call("item", 0)
	file.Call("arrayBuffer").Call("then", js.FuncOf(func(v js.Value, x []js.Value) any {
		data := js.Global().Get("Uint8Array").New(x[0])
		dst := make([]byte, data.Get("length").Int())
		js.CopyBytesToGo(dst, data)
		unsanitizedContentBytes, err := toPrettyJson(dst)
		if err != nil {
			errorFollowUp(err, true)
		}
		unsanitizedContent := string(unsanitizedContentBytes)
		filePath := file.Get("name").String()
		//fileContent := js.Global().Call("readFileContent")
		println("File has been chosen: ", filePath)

		fileExtension := filepath.Ext(filePath)[1:]
		println("Rule sets available: ", len(ruleSets))
		sanitizedContent, err := Sanitize(unsanitizedContent, fileExtension, ruleSets, config)
		if err != nil {
			errorFollowUp(err, false)
		}
		js.Global().Call(
			"showOutput",
			filePath,
			unsanitizedContent,
			generateSanitizedFileName(filePath),
			sanitizedContent,
			getRuleFilePath(fileExtension),
		)
		return nil
	}))

	return nil
}

// Start of execution.
func main() {
	// Load config.
	_, err := getResponse("script/config.json", &config)
	if err != nil {
		errorFollowUp(err, true)
	}

	// Load rule sets.
	allowedFileFormats := ""
	for _, supportedFileExtension := range config.SupportedFileExtensions {
		println("Loading rule set for " + supportedFileExtension + " files.")
		ruleSetStruct := RuleSet{}
		_, err := getResponse(getRuleFilePath(supportedFileExtension), &ruleSetStruct)

		if err != nil {
			errorFollowUp(err, false)
		} else {
			ruleSets[supportedFileExtension] = ruleSetStruct
			if allowedFileFormats != "" {
				allowedFileFormats += ","
			}
			allowedFileFormats += "." + supportedFileExtension
		}
	}
	println("Allowed file formats: ", allowedFileFormats)
	uploadButton := document.Call("getElementById", "upload_button")
	// Restricts the file types that can be loaded based on rule set availability.
	uploadButton.Call("setAttribute", "accept", allowedFileFormats)
	// Set the callback to invoke when a file is selected.
	uploadButton.Set("oninput", js.FuncOf(sanitizeCallback))

	// Keep the script running for callbacks to be processed.
	select {}
}

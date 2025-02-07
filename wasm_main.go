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
	"strconv"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

var jsGlobal = js.Global()
var jsCall = jsGlobal.Call

type RuleDetectionTaskInput struct {
	Content      *string
	RuleJsonPath string
	RuleInfo     RuleInfo
	Config       *Config
}

// Generic method to run tasks in parallel.
func runTasks[I any, O any](task func(I, *chan O, *sync.WaitGroup), taskInputs *[]I) *[]O {
	tasksCount := len(*taskInputs)
	waitGroup := sync.WaitGroup{}
	taskOutputs := make([]O, tasksCount)
	channel := make(chan O, tasksCount)
	for _, taskInput := range *taskInputs {
		// We add 1 to the wait group. Each worker will decrease it by 1 once it's done.
		waitGroup.Add(1)

		// Spawn a goroutine
		go task(taskInput, &channel, &waitGroup)
	}
	// Now we wait for all tasks to finish.
	waitGroup.Wait()

	// Close the channel or the following loop will get stuck.
	close(channel)

	for taskOutput := range channel {
		taskOutputs = append(taskOutputs, taskOutput)
	}
	return &taskOutputs
}

type Config struct {
	MaximumInputFileSizeThroughWebsiteInMB int      `json:"MaximumInputFileSizeThroughWebsiteInMB"`
	MaximumInputFilesThroughWebsite        int      `json:"MaximumInputFilesThroughWebsite"`
	RemovedSecretReplacement               string   `json:"RemovedSecretReplacement"`
	SecretPrefix                           string   `json:"SecretPrefix"`
	SupportedFileExtensions                []string `json:"SupportedFileExtensions"`
	SupportedActions                       []string `json:"SupportedActions"`
}

type RuleInfo struct {
	Description string `yaml:"description"`
	Action      string `yaml:"action"`
}

type RuleSet struct {
	Description string              `yaml:"description"`
	Format      string              `yaml:"format"`
	Rules       map[string]RuleInfo `yaml:"rules"`
}

var config = Config{}
var ruleSets = map[string]RuleSet{}
var document = jsGlobal.Get("document")

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
	println("url = ", url)
	response, err := http.Get(url)
	if response == nil || response.StatusCode != 200 || err != nil {
		return nil, err
	}
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = response.Body.Close()
	if err == nil && responseStruct != nil {
		contentType := response.Header.Get("Content-Type")
		fmt.Println("Content-Type = ", contentType)
		fmt.Println("Body = ", string(bodyBytes))
		if strings.Contains(contentType, "application/json") {
			err = json.Unmarshal(bodyBytes, responseStruct)
		} else if strings.Contains(contentType, "application/yaml") || strings.Contains(contentType, "text/yaml") {
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

func sanitizeFileTask(file js.Value, errorsChannel *chan error, waitGroup *sync.WaitGroup) {
	var err error = nil
	file.Call("arrayBuffer").Call("then", js.FuncOf(func(v js.Value, x []js.Value) any {
		data := jsGlobal.Get("Uint8Array").New(x[0])
		dst := make([]byte, data.Get("length").Int())
		js.CopyBytesToGo(dst, data)
		filePath := file.Get("name").String()
		fileExtension := filepath.Ext(filePath)[1:]
		unsanitizedContentBytes, err := toPrettyJson(dst)
		if err != nil {
			jsCall("resetPageAfterAlert", "Error parsing '"+filePath+"' : "+err.Error())
			errorFollowUp(err, false)
			return nil
		}
		unsanitizedContent := string(unsanitizedContentBytes)
		println("Rule sets available: ", len(ruleSets))
		sanitizedFileName := generateSanitizedFileName(filePath)
		sanitizedContent, diffPatchText, isDiffEmpty, err := Sanitize(unsanitizedContent, fileExtension, filePath, sanitizedFileName, ruleSets, config)
		if err != nil {
			errorFollowUp(err, false)
		}
		println("Showing output. filePath=", filePath, ", time=", time.Now().Unix())
		jsCall(
			"addOutput",
			filePath,
			unsanitizedContent,
			sanitizedFileName,
			sanitizedContent,
			diffPatchText,
			isDiffEmpty,
			getRuleFilePath(fileExtension),
		)
		return nil
	}))
	*errorsChannel <- err
	waitGroup.Done()
}

func sanitizeCallbackFromJS(_ js.Value, _ []js.Value) any {
	/**
	Callback when an input file is selected to be sanitized.
	*/
	uploadButton := document.Call("getElementById", "upload_button")
	filesElement := uploadButton.Get("files")
	filesCount := filesElement.Get("length").Int()
	if filesCount <= 0 {
		return nil
	} else if filesCount > config.MaximumInputFilesThroughWebsite {
		jsCall("resetPageAfterAlert", "Cannot sanitize more than "+strconv.Itoa(config.MaximumInputFilesThroughWebsite)+" files at a time.\nSelect a lesser number of files.")
	} else {
		jsCall("clearOutputs")
		jsCall("showElement", "display_panel")
		jsCall("showElement", "overlay-spinner", "flex")
		files := make([]js.Value, filesCount)
		filesIterated := map[string]int{}
		for index := 0; index < filesCount; index++ {
			file := filesElement.Call("item", index)
			fileSizeInMB := file.Get("size").Int() / (1024 * 1024)
			filePath := file.Get("name").String()
			if fileSizeInMB > config.MaximumInputFileSizeThroughWebsiteInMB {
				jsCall("resetPageAfterAlert", "Size of file "+filePath+" ("+strconv.Itoa(fileSizeInMB)+" MB) exceeds maximum supported file size of "+strconv.Itoa(config.MaximumInputFileSizeThroughWebsiteInMB)+" MB.")
				return nil
			}
			if _, isPresent := filesIterated[filePath]; isPresent {
				jsCall("resetPageAfterAlert", "Multiple files with the same name ("+filePath+") isn't supported. Please choose files with different names.")
				return nil
			}
			filesIterated[filePath] = 1
			files[index] = file
		}
		_ = runTasks(sanitizeFileTask, &files)
	}
	return nil
}

// Start of execution.
func main() {
	// Load config.
	_, err := getResponse("script/config.json", &config)
	if err != nil {
		errorFollowUp(err, true)
	}

	println("SupportedFileExtensions = ", strings.Join(config.SupportedFileExtensions, ","))

	// Load rule sets.
	allowedFileFormats := ""
	for _, supportedFileExtension := range config.SupportedFileExtensions {
		println("Loading rule set for " + supportedFileExtension + " files.")
		ruleSetStruct := RuleSet{}
		_, err := getResponse(getRuleFilePath(supportedFileExtension), &ruleSetStruct)

		if err != nil {
			println("Error loading rule set for " + supportedFileExtension + " files.")
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
	// Set the callback to invoke when a file is selected.
	uploadButton.Set("oninput", js.FuncOf(sanitizeCallbackFromJS))
	// Restricts the file types that can be loaded based on rule set availability.
	uploadButton.Call("setAttribute", "accept", allowedFileFormats)

	// Keep the script running for callbacks to be processed.
	select {}
}

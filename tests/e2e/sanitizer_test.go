package e2e

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"github.com/tebeka/selenium/firefox"
)

var contextualReplacement = "RANDOM"
var currentPath, _ = filepath.Abs(".")
var downloadPath = filepath.Join(currentPath, "tmp")
var server *http.Server
var rulesFile = "rules/har.yaml"
var seleniumPort = 4444
var sanitizeFileToExpectedSanitation = map[string][][]string{}
var changedLines = map[string]map[int]string{}

/**
Example formats
sanitizeFileToExpectedSanitation = {
	fileToSanitizeName1: [lineBeforeSanitization1, lineAfterSanitization1], [lineBeforeSanitization2, lineAfterSanitization2], ...],
	fileToSanitizeName2: [[...], ...],
	...
}

changedLines = {
	sanitizedFileName1: {
		lineNum1: lineAfterSanitization1,
		lineNum2: lineAfterSanitization2,
		...
	},
	sanitizedFileName2: {...},
}
*/

func prepareMapTestScenario() {
	sanitizeFileToExpectedSanitation["github.com.har"] = [][]string{{"password12345", "<REMOVED>"}}

}

func getChromeDriver() (*selenium.Service, selenium.WebDriver) {

	service, err := selenium.NewChromeDriverService("chromedriver", seleniumPort)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// configure the browser options
	caps := selenium.Capabilities{}
	prefs := make(map[string]interface{})
	prefs["download.default_directory"] = downloadPath
	caps.AddChrome(chrome.Capabilities{Prefs: prefs})

	webDriver := setAndGetDriver(caps, "")

	return service, webDriver
}

func getFirefoxDriver() (*selenium.Service, selenium.WebDriver) {

	service, err := selenium.NewGeckoDriverService("geckodriver", seleniumPort)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// configure the browser options
	caps := selenium.Capabilities{}
	prefs := make(map[string]interface{})
	prefs["browser.download.dir"] = downloadPath
	prefs["browser.download.folderList"] = 2

	caps.AddFirefox(firefox.Capabilities{Prefs: prefs})

	webDriver := setAndGetDriver(caps, fmt.Sprintf("http://localhost:%d", seleniumPort))
	return service, webDriver

}

func setAndGetDriver(capabilities selenium.Capabilities, urlPrefix string) selenium.WebDriver {

	driver, err := selenium.NewRemote(capabilities, urlPrefix)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal("Error (caps):", err)
		os.Exit(1)
	}

	// maximize the current window to avoid responsive rendering
	err = driver.MaximizeWindow("")
	if err != nil {
		log.Fatal("Error (Window):", err)
		os.Exit(1)
	}

	err = driver.Get("http://localhost:3000")
	if err != nil {
		log.Fatal("Error (Get):", err)
		os.Exit(1)
	}

	return driver
}

func runHtmlServer() {
	server = &http.Server{
		Addr: ":3000",
	}

	http.Handle("/", http.FileServer(http.Dir("../../")))
	mime.AddExtensionType(".yaml", "text/yaml")

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
		os.Exit(1)
	}

}

func setUp() {
	go runHtmlServer()

}

func tearDown() {

	err := os.RemoveAll(downloadPath)
	if err != nil {
		log.Fatalf("Error deleting download folder: %s", err)
		os.Exit(1)
	}
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err = server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %s", err)
		os.Exit(1)
	}

}

func uploadFiles(webDriver selenium.WebDriver, filesToSanitize []string) {
	resourcesPath := "./resources/hars/"
	fmt.Print(webDriver)
	productElement, err := webDriver.FindElement(selenium.ByID, "upload_button")
	if err != nil {
		log.Fatal("Error finding upload button:", err)
		os.Exit(1)
	}
	var allAbsolutePaths strings.Builder

	for i, fileToSanitize := range filesToSanitize {
		absPath, err := filepath.Abs(resourcesPath + fileToSanitize)
		if err != nil {
			log.Fatalf("Error retrieving absolute path for file %s: %s", fileToSanitize, err)
		}
		allAbsolutePaths.WriteString(absPath)
		if i != len(filesToSanitize)-1 {
			allAbsolutePaths.WriteString(" \n ")
		}
	}
	fmt.Println("absolute paths:", allAbsolutePaths.String())
	productElement.SendKeys(allAbsolutePaths.String())
	productElement.Click()
}

func verifyFileSanitation(t *testing.T, fileDivElement selenium.WebElement) {
	contextualReplacementMapping := make(map[string]string)

	fileNameElement, _ := fileDivElement.FindElement(selenium.ByCSSSelector, ".d2h-file-name")
	fileNameElementText, _ := fileNameElement.Text()
	fileNameElementFileNames := strings.Split(fileNameElementText, " → ")

	fmt.Println("filename = ", fileNameElementFileNames)
	originalFileName := fileNameElementFileNames[0]
	sanitizedFileName := fileNameElementFileNames[1]
	fmt.Println("Original filename = ", fileNameElementFileNames[0])
	changedLines[sanitizedFileName] = make(map[int]string)
	sanitizedFileChangeLines := changedLines[sanitizedFileName]

	expectedValues, ok := sanitizeFileToExpectedSanitation[originalFileName]

	if !ok {
		t.Fatal("Could not find sanitized file name (", originalFileName, ") in map. This shouldn't happen...")
	}

	deletedElements, err := fileDivElement.FindElements(selenium.ByTagName, "del")

	if err != nil {
		t.Fatal("Could not find deleted elements", err)
	}
	insertedElements, err := fileDivElement.FindElements(selenium.ByTagName, "ins")

	if err != nil {
		t.Fatal("Could not find inserted elements", err)
	}

	lineNumElements, err := fileDivElement.FindElements(selenium.ByCSSSelector, ".d2h-code-linenumber.d2h-del.d2h-change")

	if err != nil {
		t.Fatal("Could not find line numbers", err)
	}

	assert.Equal(t, len(deletedElements), len(insertedElements), "Deleted Elements length is different from inserted elements: %d vs. %d", len(deletedElements), len(insertedElements))

	for index, deletedElement := range deletedElements {
		fmt.Printf("Deleted Elements: %d\nInserted Elements: %d\n", len(deletedElements), len(insertedElements))
		deletedElementText, err := deletedElement.Text()

		if err != nil {
			t.Fatal(err)
		}

		insertedElementText, err := insertedElements[index].Text()

		if err != nil {
			t.Fatal(err)
		}
		expectedDeletedValue := expectedValues[index][0]
		expectedInsertedValue := expectedValues[index][1]

		fmt.Printf("Deleted Elements: %s\nInserted Elements: %s\n", deletedElementText, insertedElementText)
		assert.Equal(t, expectedDeletedValue, deletedElementText, "Did not delete expected parameter (%s), but removed {%s}", expectedDeletedValue, deletedElementText)

		if expectedInsertedValue == contextualReplacement {
			mappedInsertedValue, ok := contextualReplacementMapping[expectedDeletedValue]
			if !ok {
				contextualReplacementMapping[expectedDeletedValue] = expectedInsertedValue
			} else {
				expectedInsertedValue = mappedInsertedValue
			}
		}
		assert.Equal(t, expectedInsertedValue, insertedElementText, "Expected inserted value is %s, but got %s", expectedInsertedValue, insertedElementText)

		lineNum1Element, err := lineNumElements[index].FindElement(selenium.ByCSSSelector, ".line-num1")
		if err != nil {
			if err != nil {
				t.Fatal(err)
			}
		}
		lineNum1Value, err := lineNum1Element.Text()

		if err != nil {
			t.Fatal(err)
		}

		lineNum, err := strconv.Atoi(lineNum1Value)

		if err != nil {
			t.Fatalf("Could not convert number %s: %s", lineNum1Value, err)
		}

		sanitizedFileChangeLines[lineNum] = expectedInsertedValue
	}
}

func testLogic(t *testing.T, webDriver selenium.WebDriver) {
	filesToSanitize := make([]string, len(sanitizeFileToExpectedSanitation))

	i := 0
	for fileToSanitize := range sanitizeFileToExpectedSanitation {
		filesToSanitize[i] = fileToSanitize
	}

	time.Sleep(5 * time.Second)
	uploadFiles(webDriver, filesToSanitize)
	time.Sleep(20 * time.Second)
	values, err := webDriver.FindElements(selenium.ByCSSSelector, ".sanitized_diff_div")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("len of sanitized_diff_div", len(values))
	for _, fileDivElement := range values {
		verifyFileSanitation(t, fileDivElement)
	}

	// Check download
	downloadButton, _ := webDriver.FindElement(selenium.ByID, "download_button_label")
	downloadButton.Click()
	time.Sleep(25 * time.Second)

	for sanitizedFileName := range changedLines {
		expectedFileToDownloadPath := filepath.Join(downloadPath, sanitizedFileName)
		_, downloadedFileError := os.Stat(expectedFileToDownloadPath)
		assert.Nil(t, downloadedFileError, downloadedFileError)
		verifyDownloadedFile(t, expectedFileToDownloadPath, changedLines[sanitizedFileName])
	}

	// View Rules
	verifyViewRules(t, webDriver)

}

func verifyDownloadedFile(t *testing.T, filename string, changedLines map[int]string) {

	inputFile, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}

	defer inputFile.Close()
	lineNum := 0
	sc := bufio.NewScanner(inputFile)
	for sc.Scan() {
		lineNum++
		changedLine, ok := changedLines[lineNum]
		if ok {
			actualLine := sc.Text()
			assert.Contains(t, actualLine, changedLine, "Did not download sanitized file. Expected %s for line %d but got %s", changedLine, lineNum, actualLine)
		}
	}
}

func verifyViewRules(t *testing.T, webDriver selenium.WebDriver) {
	viewRulesElement, err := webDriver.FindElement(selenium.ByID, "view_rules_button")

	if err != nil {
		t.Fatal(err)
	}

	viewRulesElement.Click()
	time.Sleep(3 * time.Second)
	handles, err := webDriver.WindowHandles()

	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, len(handles), "No new window was opened. Expected 2 but got %d", len(handles))

	// assert there's two windows
	webDriver.SwitchWindow(handles[1])
	url, _ := webDriver.CurrentURL()
	fmt.Println(url)
	assert.Contains(t, url, rulesFile, "Did not get the right url. Expected: https://locathost:3000/%s but got %s", rulesFile, url)

	fmt.Println(handles)
}

func resetEnvironment() {
	_, err := os.Stat(downloadPath)
	if err == nil {
		err = os.RemoveAll(downloadPath)
		if err != nil {
			log.Fatal("Could not delete download path (", downloadPath, "). Failing tests...", err)
		}
	} else if !os.IsNotExist(err) {
		log.Fatal("Could not check if download path (", downloadPath, ") exists. Failing tests...", err)
	}
	err = os.Mkdir(downloadPath, 0700)

	if err != nil {
		log.Fatal("Could create download path (", downloadPath, ")", err)
	}
}

func TestFirefoxDriver(t *testing.T) {
	resetEnvironment()
	service, geckoDriver := getFirefoxDriver()
	testLogic(t, geckoDriver)
	geckoDriver.Quit()
	defer service.Stop()
}

func TestChromeDriver(t *testing.T) {
	resetEnvironment()
	service, chromeDriver := getChromeDriver()
	testLogic(t, chromeDriver)
	chromeDriver.Quit()
	defer service.Stop()
}

func TestInvalidHarFile(t *testing.T) {
	resetEnvironment()
	service, chromeDriver := getChromeDriver()
	filesToSanitize := []string{"invalid_har"}
	uploadFiles(chromeDriver, filesToSanitize)
	// check that alert is raised
	chromeDriver.Quit()
	defer service.Stop()
}

func TestMain(m *testing.M) {
	setUp()
	prepareMapTestScenario()
	m.Run()
	tearDown()
}

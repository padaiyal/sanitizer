package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
var fileToSanitize = "./resources/hars/github.com.har"
var expectedFileToDownload = filepath.Join(downloadPath, "githubcom_sanitized.har")
var selenium_port = 4444

func getChromeDriver() (*selenium.Service, selenium.WebDriver) {

	service, err := selenium.NewChromeDriverService("./resources/drivers/chromedriver", selenium_port)
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

	service, err := selenium.NewGeckoDriverService("./resources/drivers/geckodriver", selenium_port)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// configure the browser options
	caps := selenium.Capabilities{}
	prefs := make(map[string]interface{})
	prefs["browser.download.dir"] = downloadPath
	prefs["browser.download.folderList"] = 2

	caps.AddFirefox(firefox.Capabilities{Prefs: prefs})

	webDriver := setAndGetDriver(caps, fmt.Sprintf("http://localhost:%d", selenium_port))
	return service, webDriver

}

func getEdgeWebDriver() (*selenium.Service, selenium.WebDriver) {

	service, err := selenium.NewChromeDriverService("./resources/drivers/msedgedriver", selenium_port)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// configure the browser options
	caps := selenium.Capabilities{}
	prefs := make(map[string]interface{})
	// prefs["download.default_directory"] = downloadPath
	prefs["savefile.default_directory"] = downloadPath
	caps.AddChrome(chrome.Capabilities{Prefs: prefs})

	webDriver := setAndGetDriver(caps, "")

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

	http.Handle("/", http.FileServer(http.Dir("../")))

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
		os.Exit(1)
	}

}

func setUp() {
	go runHtmlServer()
	os.Mkdir(downloadPath, os.FileMode(0522))

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

func uploadFile(webDriver selenium.WebDriver, fileToSanitize string) {
	fmt.Print(webDriver)
	productElement, err := webDriver.FindElement(selenium.ByID, "upload_button")
	if err != nil {
		log.Fatal("Error finding upload button:", err)
		os.Exit(1)
	}

	absPath, err := filepath.Abs(fileToSanitize)
	if err != nil {
		log.Fatalf("Error retrieving absolute path for file %s: %s", fileToSanitize, err)
	}

	productElement.SendKeys(absPath)
	productElement.Click()
}

func testLogic(t *testing.T, webDriver selenium.WebDriver) {
	contextualReplacementMapping := make(map[string]string)
	changedLines := make(map[int]string)

	expectedDeletedElements := []string{"password12345"}
	expectedInsertedElements := []string{"<REMOVED>"}

	uploadFile(webDriver, fileToSanitize)

	deletedElements, err := webDriver.FindElements(selenium.ByTagName, "del")

	if err != nil {
		t.Fatal(err)
	}
	insertedElements, err := webDriver.FindElements(selenium.ByTagName, "ins")

	if err != nil {
		t.Fatal(err)
	}

	lineNumElements, err := webDriver.FindElements(selenium.ByCSSSelector, ".d2h-code-linenumber.d2h-del.d2h-change")

	if err != nil {
		t.Fatal(err)
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

		fmt.Printf("Deleted Elements: %s\nInserted Elements: %s\n", deletedElementText, insertedElementText)
		assert.Equal(t, expectedDeletedElements[index], deletedElementText, "Did not delete expected parameter (%s), but removed {%s}", expectedDeletedElements[index], deletedElementText)

		expectedInsertedValue := expectedInsertedElements[index]
		if expectedInsertedValue == contextualReplacement {
			mappedInsertedValue, ok := contextualReplacementMapping[expectedDeletedElements[index]]
			if !ok {
				contextualReplacementMapping[expectedDeletedElements[index]] = expectedInsertedValue
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

		changedLines[lineNum] = expectedInsertedValue

		// Check download
		downloadButton, _ := webDriver.FindElement(selenium.ByID, "download_button")
		downloadButton.Click()
		time.Sleep(5 * time.Second)

		_, downloadedFileError := os.Stat(expectedFileToDownload)
		assert.Nil(t, downloadedFileError, downloadedFileError)

		verifyDownloadedFile(t, expectedFileToDownload, changedLines)

		// View Rules
		verifyViewRules(t, webDriver)
	}

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
	assert.Equal(t, 2, len(handles), "No new window was opend. Expected 2 but got %d", len(handles))

	// assert there's two windows
	webDriver.SwitchWindow(handles[1])
	url, _ := webDriver.CurrentURL()
	fmt.Println(url)
	assert.Contains(t, url, rulesFile, "Did not get the right url. Expected: https://locathost:3000/%s but got %s", rulesFile, url)

	fmt.Println(handles)
}

func resetEnvironment() {

	_, err := os.Stat(expectedFileToDownload)
	if err != nil {
		return
	}

	err = os.Remove(expectedFileToDownload)
	if err != nil {
		log.Fatalf("Error cleaning file: %v", err)
		os.Exit(1)
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

func TestEdgeDriver(t *testing.T) {
	resetEnvironment()
	service, edgeDriver := getEdgeWebDriver()

	err := edgeDriver.Get("edge://settings/profiles")

	if err != nil {
		log.Fatal("Error (Get):", err)
		os.Exit(1)
	}

	element, err := edgeDriver.FindElement(selenium.ByID, "search_input")
	if err != nil {
		log.Fatal("Error (Find):", err)
		os.Exit(1)
	}

	element.SendKeys("Downloads")
	element.Submit()
	// edge://settings/profiles
	// div id=/downloads
	// button aria-label="Change location"
	// TODO: Check this
	element, err = edgeDriver.FindElement(selenium.ByCSSSelector, "[aria-label=\"Change location\"]")
	if err != nil {
		log.Fatal("Error (Find):", err)
		os.Exit(1)
	}
	element.SendKeys(downloadPath)
	element.Click()
	time.Sleep(100)
	testLogic(t, edgeDriver)
	edgeDriver.Quit()
	defer service.Stop()

}

func TestMain(m *testing.M) {
	setUp()
	m.Run()
	tearDown()
}

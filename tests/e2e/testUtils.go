package e2e

import (
	"context"
	"errors"
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"github.com/tebeka/selenium/firefox"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const FIREFOX string = "Firefox"
const CHROME string = "Chrome"

var currentPath string
var DownloadPath string
var ResourcesPath string
var seleniumPort int
var server *http.Server
var MaxWaitTimeout time.Duration

type SanitizedLines struct {
	originalLine string
	changedLine  string
}

type SanitizedFile struct {
	originalFilename  string
	sanitizedFilename string
	lines             []SanitizedLines
}

func GetDriver(driverType string) (*selenium.Service, selenium.WebDriver, error) {
	var service *selenium.Service
	var err error
	var fullPath string

	urlPrefix := ""
	caps := selenium.Capabilities{}
	prefs := make(map[string]interface{})
	osType := runtime.GOOS
	//args := []string{"--headless"}
	args := []string{}
	path, err := filepath.Abs("../../depot/webdriver")
	if err != nil {
		return nil, nil, fmt.Errorf("error getting absolute path of depot/webdriver: %s", err)
	}

	if driverType == FIREFOX {
		fmt.Printf("BROWSER PATH: %s", os.Getenv("FIREFOX_BROWSER_PATH"))
		//fullPath = filepath.Join(currentPath, path, osType, "geckodriver")
		fullPath = filepath.Join(path, fmt.Sprintf("geckodriver_%s_%s", osType, os.Getenv("GECKO_DRIVER_VERSION")))
		fmt.Printf("This is the driver path: %s", fullPath)
		service, err = selenium.NewGeckoDriverService(fullPath, seleniumPort)
		prefs["browser.download.dir"] = DownloadPath
		prefs["browser.download.folderList"] = 2

		caps.AddFirefox(firefox.Capabilities{Binary: os.Getenv("FIREFOX_BROWSER_PATH"), Prefs: prefs, Args: args})

		urlPrefix = fmt.Sprintf("http://localhost:%d", seleniumPort)

	} else if driverType == CHROME {
		fmt.Printf("BROWSER PATH: %s", os.Getenv("CHROME_BROWSER_PATH"))
		//fullPath = filepath.Join(currentPath, path, osType, "chromedriver")
		fullPath = filepath.Join(path, fmt.Sprintf("chromedriver_%s_%s", osType, os.Getenv("CHROME_DRIVER_VERSION")))
		service, err = selenium.NewChromeDriverService(fullPath, seleniumPort)
		prefs["download.default_directory"] = DownloadPath
		prefs["profile.default_content_setting_values.automatic_downloads"] = 1
		fmt.Printf("This is the driver path: %s", fullPath)
		//args = append(args, "--disable-dev-shm-usage")
		args = append(args, "--no-sandbox")
		caps.AddChrome(chrome.Capabilities{Path: os.Getenv("CHROME_BROWSER_PATH"), Prefs: prefs, Args: args})

	} else {
		return nil, nil, fmt.Errorf("unsupported driver type: %s", driverType)
	}

	driver, err := selenium.NewRemote(caps, urlPrefix)
	if err != nil {
		log.Fatal("Error (caps):", err)
	}

	// maximize the current window to avoid responsive rendering
	err = driver.MaximizeWindow("")
	if err != nil {
		log.Fatal("Error (Window):", err)
	}

	err = driver.Get("http://localhost:3000")
	if err != nil {
		log.Fatal("Error (Get):", err)
	}

	return service, driver, nil
}

func RunHtmlServer() {
	server = &http.Server{
		Addr: ":3000",
	}

	http.Handle("/", http.FileServer(http.Dir("../../")))
	mime.AddExtensionType(".yaml", "text/yaml")

	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server error: %v", err)
	}
}

func SetUp() {
	var err error

	currentPath, err = filepath.Abs(".")
	if err != nil {
		log.Fatal("Could not get absolute path of current directory", err)
	}

	DownloadPath = filepath.Join(currentPath, "tmp")
	ResourcesPath = filepath.Join(currentPath, "resources/hars/")
	seleniumPort = 4444
	MaxWaitTimeout = 90 * time.Second

	go RunHtmlServer()
}

func TearDown() {
	err := os.RemoveAll(DownloadPath)
	if err != nil {
		log.Fatalf("Error deleting download folder: %s", err)
	}
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err = server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %s", err)
	}
}

func ResetEnvironment() {
	log.Println("Resetting environment...")
	_, err := os.Stat(DownloadPath)
	if err == nil {
		err = os.RemoveAll(DownloadPath)
		if err != nil {
			log.Fatal("Could not delete download path (", DownloadPath, "). Failing tests...", err)
		}
	} else if !os.IsNotExist(err) {
		log.Fatal("Could not check if download path (", DownloadPath, ") exists. Failing tests...", err)
	}
	err = os.Mkdir(DownloadPath, 0700)

	if err != nil {
		log.Fatal("Could create download path (", DownloadPath, ")", err)
	}
}

func RunE2ETest(browser string, t *testing.T, testLogic func(t *testing.T, driver selenium.WebDriver) error) {
	var service *selenium.Service
	var driver selenium.WebDriver
	var err error

	ResetEnvironment()
	service, driver, err = GetDriver(browser)
	if err != nil {
		log.Fatalf("Error getting driver: %s", err)
	}

	testError := testLogic(t, driver)
	err = driver.Close()
	if err != nil {
		t.Log("Error closing driver:", err)
	}

	err = driver.Quit()
	if err != nil {
		t.Log("Error quitting driver:", err)
	}

	if testError != nil {
		log.Fatalf("Error running test: %s", testError)
	}
	defer func(service *selenium.Service) {
		err := service.Stop()
		if err != nil {
			t.Log("Error stopping service:", err)
		}
	}(service)
}

func UploadFiles(webDriver selenium.WebDriver, filesToSanitize []string) error {

	fmt.Print(webDriver)
	productElement, err := webDriver.FindElement(selenium.ByID, "upload_button")
	if err != nil {
		log.Fatal("Error finding upload button:", err)
	}
	var allAbsolutePaths strings.Builder

	for i, fileToSanitize := range filesToSanitize {
		absPath := filepath.Join(ResourcesPath, fileToSanitize)
		allAbsolutePaths.WriteString(absPath)
		if i != len(filesToSanitize)-1 {
			allAbsolutePaths.WriteString("\n")
		}
	}
	fmt.Println("absolute paths:", allAbsolutePaths.String())
	err = productElement.SendKeys(allAbsolutePaths.String())
	if err != nil {
		log.Fatal("Error Sending file paths:", err)
	}
	return nil
}

func IsUploadButtonReady(webDriver selenium.WebDriver) (bool, error) {
	uploadButtonElement, _ := webDriver.FindElement(selenium.ByID, "upload_button")
	if uploadButtonElement != nil {

		isReady, err := uploadButtonElement.GetAttribute("accept")
		// This means that the wasm script hasn't loaded yet...
		if err != nil && strings.Contains(err.Error(), "nil return value") {
			err = nil
		}
		return len(isReady) > 0, err
	}
	return false, nil
}

func WaitForUploadButtonIsReady(webDriver selenium.WebDriver, timeout time.Duration) error {
	err := webDriver.WaitWithTimeout(IsUploadButtonReady, timeout)

	if err != nil {
		return fmt.Errorf("upload button isn't ready after %v seconds, Error: %s", timeout, err)
	}
	return nil
}

func isUntouchedFilesReady(webDriver selenium.WebDriver) (bool, error) {
	unsanitizedElements, _ := webDriver.FindElements(selenium.ByID, "unsanitized_file_p")
	if len(unsanitizedElements) > 0 {
		return true, nil
	}
	return false, nil
}

func WaitForUntouchedFilesReady(webDriver selenium.WebDriver, timeout time.Duration) error {
	err := webDriver.WaitWithTimeout(isUntouchedFilesReady, 90*time.Second)
	if err != nil {
		return fmt.Errorf("untouched Files is not displayed after %v seconds, Error: %s", timeout, err)
	}
	return nil
}

func WaitForNewWindowsOpen(webDriver selenium.WebDriver, windowsToWaitNumber int, timeout time.Duration) error {
	err := webDriver.WaitWithTimeout(func(wd selenium.WebDriver) (bool, error) {
		handles, _ := wd.WindowHandles()
		if len(handles) == windowsToWaitNumber {
			return true, nil
		}
		return false, nil
	}, timeout)
	if err != nil {
		return fmt.Errorf("number of expected windows did not open after %v seconds. Error: %s", timeout, err)
	}
	return nil
}

func GetRemoveOrInsertInDiff(diffs []diffmatchpatch.Diff) []diffmatchpatch.Diff {
	var nonEqualDiffs []diffmatchpatch.Diff
	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffDelete, diffmatchpatch.DiffInsert:
			nonEqualDiffs = append(nonEqualDiffs, diff)
		default:

		}
	}
	return nonEqualDiffs
}

func CreateMappingOfDeletedAndInsertedStringsInDiff(diffs []diffmatchpatch.Diff) []SanitizedLines {
	var sanitizedLines []SanitizedLines
	sanitizedLine := SanitizedLines{}

	for _, diff := range diffs {

		if diff.Type == diffmatchpatch.DiffDelete && len(sanitizedLine.originalLine) > 0 && len(sanitizedLine.changedLine) > 0 {
			sanitizedLines = append(sanitizedLines, sanitizedLine)
			sanitizedLine = SanitizedLines{}
		}

		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			sanitizedLine.originalLine += diff.Text
		case diffmatchpatch.DiffInsert:
			sanitizedLine.changedLine += diff.Text
		default:
		}

	}
	sanitizedLines = append(sanitizedLines, sanitizedLine)
	return sanitizedLines
}

func PopulateSanitizedFileInfoStruct(originalFileName string, SanitizedFileName string, diffs []diffmatchpatch.Diff) SanitizedFile {
	return SanitizedFile{
		originalFilename:  originalFileName,
		sanitizedFilename: SanitizedFileName,
		lines:             CreateMappingOfDeletedAndInsertedStringsInDiff(diffs),
	}
}

func CreateSanitizedFileName(originalFileName string) string {
	return strings.Replace(originalFileName, ".har", "_sanitized.har", 1)
}

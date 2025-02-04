package e2e

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/stretchr/testify/assert"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
	"github.com/tebeka/selenium/firefox"
)

const FIREFOX string = "firefox"
const CHROME string = "chrome"

var currentPath string
var DownloadPath string
var ResourcesPath string
var seleniumPort int
var server *http.Server
var MaxWaitTimeout time.Duration
var rulesFile = "rules/har.yaml"

type SanitizedLines struct {
	originalLine string
	changedLine  string
}

type SanitizedFile struct {
	originalFilename  string
	sanitizedFilename string
	lines             []SanitizedLines
}

/*
Generic methods for running the tests
*/

func GetDriver(driverType string) (*selenium.Service, selenium.WebDriver, error) {
	var service *selenium.Service
	var err error
	var fullPath string

	urlPrefix := ""
	caps := selenium.Capabilities{}
	prefs := make(map[string]interface{})
	osType := runtime.GOOS
	args := []string{"--headless"}
	// Uncomment to test locally
	//args = []string{}
	path, err := filepath.Abs("../../depot/webdriver")
	if err != nil {
		return nil, nil, fmt.Errorf("error getting absolute path of depot/webdriver: %s", err)
	}

	if driverType == FIREFOX {

		fullPath = filepath.Join(path, fmt.Sprintf("geckodriver_%s_%s", osType, os.Getenv("GECKO_DRIVER_VERSION")))
		service, err = selenium.NewGeckoDriverService(fullPath, seleniumPort)
		prefs["browser.download.dir"] = DownloadPath
		prefs["browser.download.folderList"] = 2

		caps.AddFirefox(firefox.Capabilities{Binary: os.Getenv("FIREFOX_BROWSER_PATH"), Prefs: prefs, Args: args})

		urlPrefix = fmt.Sprintf("http://localhost:%d", seleniumPort)

	} else if driverType == CHROME {
		fullPath = filepath.Join(path, fmt.Sprintf("chromedriver_%s_%s", osType, os.Getenv("CHROME_DRIVER_VERSION")))
		service, err = selenium.NewChromeDriverService(fullPath, seleniumPort)
		prefs["download.default_directory"] = DownloadPath
		prefs["profile.default_content_setting_values.automatic_downloads"] = 1

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
	log.Println("Setting up the Server")

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

func CloseWebDriverAndService(webDriver selenium.WebDriver, service *selenium.Service) {
	var err error

	err = webDriver.Close()
	if err != nil {
		log.Println("Error closing driver:", err)
	}

	err = webDriver.Quit()
	if err != nil {
		if strings.Contains(err.Error(), "invalid session id: Tried to run command without establishing a connection") {
			log.Println("Warning: ", err)
		} else {
			log.Println("Error quitting driver:", err)
		}
	}

	defer func(service *selenium.Service) {
		err := service.Stop()
		if err != nil {
			log.Println("Error stopping service:", err)
		}
	}(service)
}

/*
Methods for interacting with the UI
*/

func UploadFiles(webDriver selenium.WebDriver, filesToSanitize []string) error {

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

	err = productElement.SendKeys(allAbsolutePaths.String())
	if err != nil {
		log.Fatal("Error Sending file paths:", err)
	}
	return nil
}

/*
Waiter methods
*/

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
	unSanitizedElements, _ := webDriver.FindElements(selenium.ByID, "unsanitized_file_p")
	if len(unSanitizedElements) > 0 {
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

/*
Helpers
*/

func GetRemoveOrInsertInDiff(diffs []diffmatchpatch.Diff) []diffmatchpatch.Diff {
	// Only gets the differences in a diff (so no equal values)
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
	// Convert the diffs into a slice with SanitizedLines struct
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

func verifyFileSanitizationInUI(t *testing.T, fileDivElement selenium.WebElement, sanitizedFileInfos map[string]SanitizedFile) error {
	// checks that the file sanitization diffs in the UI matches the expected sanitized file diffs

	// Asserts we are comparing the right files
	fileNameElement, _ := fileDivElement.FindElement(selenium.ByCSSSelector, ".d2h-file-name")
	fileNameElementText, _ := fileNameElement.Text()
	fileNameElementFileNames := strings.Split(fileNameElementText, " → ")

	t.Log("Original filename = ", fileNameElementFileNames[0])
	sanitizedFileInfo, ok := sanitizedFileInfos[fileNameElementFileNames[0]]
	if !ok {
		t.Fatal("SanitizedFileInfo not found")
	}
	assert.Equal(t, sanitizedFileInfo.sanitizedFilename, fileNameElementFileNames[1])

	// Getting the deleted and inserted elements from the ui, and make sure they are equal in number
	deletedElements, err := fileDivElement.FindElements(selenium.ByCSSSelector, "td[class^='d2h-del d2h-change']")
	if err != nil {
		t.Fatal("Could not find deleted elements", err)
	}

	insertedElements, err := fileDivElement.FindElements(selenium.ByCSSSelector, "td[class^='d2h-ins d2h-change']")

	if err != nil {
		t.Fatal("Could not find inserted elements", err)
	}

	assert.Equal(t, len(deletedElements), len(insertedElements), "Deleted Elements length is different from inserted elements: %d vs. %d", len(deletedElements), len(insertedElements))

	// for each diff found in UI, we check that both deleted and inserted texts matches the expected text
	for index, deletedElement := range deletedElements {

		deletedWithValueElement, err := deletedElement.FindElement(selenium.ByTagName, "del")
		if err != nil {
			return err
		}

		deletedElementText, err := deletedWithValueElement.Text()

		insertedWithValueElement, err := insertedElements[index].FindElement(selenium.ByTagName, "ins")
		if err != nil {
			return err
		}

		insertedElementText, err := insertedWithValueElement.Text()
		if err != nil {
			return err
		}

		// Removing the next line char as the UI doesn't provide the char. We are trying to verify the actual content of the string match the actual sanitization
		expectedDeletedValue := sanitizedFileInfo.lines[index].originalLine
		expectedDeletedValue = strings.Replace(expectedDeletedValue, "\n", "", -1)

		expectedInsertedValue := sanitizedFileInfo.lines[index].changedLine
		expectedInsertedValue = strings.Replace(expectedInsertedValue, "\n", "", -1)

		assert.Equal(t, expectedDeletedValue, deletedElementText, "Did not delete expected parameter (%s), but removed {%s}", expectedDeletedValue, deletedElementText)

		assert.Equal(t, expectedInsertedValue, insertedElementText, "Expected inserted value is %s, but got %s", expectedInsertedValue, insertedElementText)
	}
	return nil
}

/*
Gets the expected UI changes based on the diff between the original file and the sanitized file.
*/
func getExpectedUIDiffsFromMatchingSanitizedFileContent(t *testing.T, inputSanitizedFileNamePath string, expectedSanitizedFileNamePath string) ([]diffmatchpatch.Diff, error) {
	inputSanitizedFileContentRaw, err := os.ReadFile(inputSanitizedFileNamePath)
	if err != nil {
		t.Errorf("Could not open file %s for reading: %s", inputSanitizedFileNamePath, err)
		return nil, err
	}
	inputSanitizedFileContent := string(inputSanitizedFileContentRaw)

	matchingExpectedSanitizedFileContentRaw, err := os.ReadFile(expectedSanitizedFileNamePath)
	if err != nil {
		t.Errorf("Could not open file %s for reading: %s", expectedSanitizedFileNamePath, err)
		return nil, err
	}
	matchingExpectedSanitizedFileContent := string(matchingExpectedSanitizedFileContentRaw)

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(inputSanitizedFileContent, matchingExpectedSanitizedFileContent, false)
	diffs = dmp.DiffCleanupSemantic(diffs)
	nonEqualDiffs := GetRemoveOrInsertInDiff(diffs)

	t.Logf("Diffs From Original: %s\n", dmp.DiffPrettyText(nonEqualDiffs))

	return nonEqualDiffs, nil

}

func verifyDownloadedFile(t *testing.T, expectedSanitizedFileName string, actualSanitizedFileName string) {
	actualSanitizedFileContentByte, err := os.ReadFile(actualSanitizedFileName)
	if err != nil {
		t.Fatalf("Could not open file %s for reading: %s", actualSanitizedFileName, err)

	}

	expectedSanitizedFileContentByte, err := os.ReadFile(expectedSanitizedFileName)
	if err != nil {
		t.Fatalf("Could not open file %s for reading: %s", expectedSanitizedFileName, err)
	}

	expectedSanitizedFileContentMd5Hash := md5.Sum(expectedSanitizedFileContentByte)
	actualSanitizedFileContentMd5Hash := md5.Sum(actualSanitizedFileContentByte)

	t.Logf("expectedSanitizedFileContentMd5Hash: %s - actualSanitizedFileContentMd5Hash: %s\n",
		hex.EncodeToString(expectedSanitizedFileContentMd5Hash[:]), hex.EncodeToString(actualSanitizedFileContentMd5Hash[:]))
	assert.Equal(t, expectedSanitizedFileContentMd5Hash, actualSanitizedFileContentMd5Hash)
}

func verifyViewRules(t *testing.T, webDriver selenium.WebDriver) error {
	viewRulesElement, err := webDriver.FindElement(selenium.ByID, "view_rules_button")
	if err != nil {
		return err
	}

	err = viewRulesElement.Click()
	if err != nil {
		return err
	}

	// assert there's two windows
	err = WaitForNewWindowsOpen(webDriver, 2, MaxWaitTimeout)
	if err != nil {
		return err
	}

	handles, err := webDriver.WindowHandles()
	if err != nil {
		return err
	}

	err = webDriver.SwitchWindow(handles[1])
	if err != nil {
		return err
	}

	url, _ := webDriver.CurrentURL()
	assert.Contains(t, url, rulesFile, "Did not get the right url. Expected: https://locathost:3000/%s but got %s", rulesFile, url)

	t.Log(handles)
	return nil
}

func uploadAndVerifyAlerts(t *testing.T, webDriver selenium.WebDriver, filesToSanitize []string, expectedAlertTextMessage string) error {
	var alertText string
	var err error
	err = WaitForUploadButtonIsReady(webDriver, MaxWaitTimeout)

	if err != nil {
		return err
	}
	err = UploadFiles(webDriver, filesToSanitize)

	// This error message may appear due to racing condition, but as we are verifying alerts here, we ignore it.
	if err != nil && strings.Contains(err.Error(), "unexpected alert open") {
		t.Log("Alert while uploading file. Expected behaviour. Ignoring...")
	} else if err != nil {
		return err
	}

	err = webDriver.WaitWithTimeout(func(driver selenium.WebDriver) (bool, error) {
		alertText, _ = webDriver.AlertText()
		if len(alertText) > 0 {
			return true, nil
		}
		return false, nil
	}, MaxWaitTimeout)

	if err != nil {
		return fmt.Errorf("alert did not appear in the specified time: %s", err)
	}

	// check the alert message contains the expected text
	assert.True(t, strings.Contains(alertText, expectedAlertTextMessage), "Expected alertText message to contain '%s' but got '%s'", expectedAlertTextMessage, alertText)
	err = webDriver.DismissAlert()
	if err != nil {
		return err
	}
	_, err = webDriver.AlertText()
	assert.True(t, strings.Contains(err.Error(), "no alert open") ||
		strings.Contains(err.Error(), "no such alert"), "Alert was not dismissed")
	return nil
}

func TestingValidE2E(t *testing.T, webDriver selenium.WebDriver) error {

	var err error
	t.Log("Starting validating e2e process...\n")
	fileNamesToSanitize := []string{"github.com.har", "contextual_replacement.har", "remove_and_contextual_replacement.har"}
	untouchedFileNames := []string{"1.har", "2.har", "3.har", "4.har", "5.har", "6.har", "already_sanitized.har"}
	sanitizedFileInfos := make(map[string]SanitizedFile)

	// Prepare the map with the expected values to compare with
	for _, fileName := range fileNamesToSanitize {
		sanitizedFileInfo := PopulateSanitizedFileInfoStruct(fileName, CreateSanitizedFileName(fileName), nil)
		sanitizedFileInfos[fileName] = sanitizedFileInfo
	}

	//Wait for page to be ready
	err = WaitForUploadButtonIsReady(webDriver, MaxWaitTimeout)
	if err != nil {
		return err
	}

	t.Log("Uploading File")
	filesToSanitizePath := append(fileNamesToSanitize, untouchedFileNames...)
	err = UploadFiles(webDriver, filesToSanitizePath)
	if err != nil {
		return err
	}

	// Wait until the page is ready. Untouched file element was chosen arbitrarily
	err = WaitForUntouchedFilesReady(webDriver, MaxWaitTimeout)
	if err != nil {
		return err
	}

	// Verify Untouched Files
	t.Logf("Verifying untouched files: %s", untouchedFileNames)
	unSanitizedElements, err := webDriver.FindElements(selenium.ByID, "unsanitized_file_p")
	if err != nil {
		return err
	}

	assert.Equal(t, len(untouchedFileNames), len(unSanitizedElements),
		"Expected untouched file names to be %d, but got %d", len(untouchedFileNames), unSanitizedElements)
	for _, unSanitizedElement := range unSanitizedElements {
		actualUnSanitizedFileName, err := unSanitizedElement.Text()
		if err != nil {
			return err
		}

		assert.True(t, slices.Contains(untouchedFileNames, actualUnSanitizedFileName),
			"Unexpected untouched file '%s' found. Expected untouched files: %s", actualUnSanitizedFileName, untouchedFileNames)
	}

	// Verify downloads
	downloadButton, _ := webDriver.FindElement(selenium.ByID, "download_button_label")
	err = downloadButton.Click()
	if err != nil {
		return err
	}
	time.Sleep(25 * time.Second)

	// For each file we first verify the downloaded file matches the expected sanitized file
	for fileName, sanitizedFileInfo := range sanitizedFileInfos {

		expectedSanitizedFileNamePath := filepath.Join(ResourcesPath, "expected_sanitized_files", sanitizedFileInfo.sanitizedFilename)

		expectedSanitizedStatInfo, expectedSanitizedFileError := os.Stat(expectedSanitizedFileNamePath)
		assert.Nil(t, expectedSanitizedFileError, expectedSanitizedFileError)

		downloadedSanitizedFileNamePath := filepath.Join(DownloadPath, sanitizedFileInfo.sanitizedFilename)
		downloadedSanitizedStatInfo, downloadedSanitizedFileError := os.Stat(downloadedSanitizedFileNamePath)
		assert.Nil(t, downloadedSanitizedFileError, downloadedSanitizedFileError)

		assert.Equalf(t, expectedSanitizedStatInfo.Size(), downloadedSanitizedStatInfo.Size(), "expected sanitized file size '%d' is different from actual sanitized file size '%d'", expectedSanitizedStatInfo.Size(), downloadedSanitizedStatInfo.Size())

		verifyDownloadedFile(t, expectedSanitizedFileNamePath, downloadedSanitizedFileNamePath)

		// We get the expected changed lines based on the downloaded file
		inputFilePath := filepath.Join(ResourcesPath, fileName)
		diffs, err := getExpectedUIDiffsFromMatchingSanitizedFileContent(t, inputFilePath, downloadedSanitizedFileNamePath)
		if err != nil {
			return err
		}
		sanitizedFileInfo.lines = CreateMappingOfDeletedAndInsertedStringsInDiff(diffs)
		sanitizedFileInfos[fileName] = sanitizedFileInfo

	}

	values, err := webDriver.FindElements(selenium.ByCSSSelector, ".d2h-file-wrapper")
	if err != nil {
		return err
	}

	for _, fileDivElement := range values {
		err = verifyFileSanitizationInUI(t, fileDivElement, sanitizedFileInfos)
		if err != nil {
			return err
		}
	}

	// View Rules
	err = verifyViewRules(t, webDriver)
	if err != nil {
		return err
	}
	return nil
}

func TestingInvalidAlertDisplayed(t *testing.T, webDriver selenium.WebDriver) error {

	filesToSanitize := []string{"invalid.har"}
	expectedAlertTextMessage := "Error parsing '" + filesToSanitize[0] + "'"
	return uploadAndVerifyAlerts(t, webDriver, filesToSanitize, expectedAlertTextMessage)
}

func TestingDuplicatedFilesToSanitize(t *testing.T, webDriver selenium.WebDriver) error {

	filesToSanitize := []string{"github.com.har", "duplicated/github.com.har"}
	expectedAlertTextMessage := "Multiple files with the same name (github.com.har) isn't supported. Please choose files with different names."
	return uploadAndVerifyAlerts(t, webDriver, filesToSanitize, expectedAlertTextMessage)
}

func TestingFileUploadExceedsSizeLimit(t *testing.T, webDriver selenium.WebDriver) error {

	filesToSanitize := []string{"big.har"}
	expectedAlertTextMessage := "Size of file big.har (53 MB) exceeds maximum supported file size of 50 MB."
	return uploadAndVerifyAlerts(t, webDriver, filesToSanitize, expectedAlertTextMessage)
}

func TestingFileUploadNumberExceedsLimit(t *testing.T, webDriver selenium.WebDriver) error {

	filesToSanitize := []string{"1.har", "2.har", "3.har", "4.har", "5.har", "6.har", "7.har", "8.har",
		"contextual_replacement.har", "github.com.har", "remove_and_contextual_replacement.har"}

	expectedAlertTextMessage := "Cannot sanitize more than 10 files at a time.\nSelect a lesser number of files."
	return uploadAndVerifyAlerts(t, webDriver, filesToSanitize, expectedAlertTextMessage)
}

package e2e

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tebeka/selenium"
)

var contextualReplacement = "RANDOM"

var rulesFile = "rules/har.yaml"

var sanitizeFileToExpectedSanitation = make(map[string][][]string)
var changedLines = map[string]map[int]string{}
var untouchedFileNames []string

func prepareMapTestScenario() {
	sanitizeFileToExpectedSanitation["github.com.har"] = [][]string{{"password12345", "<REMOVED>"}}
	sanitizeFileToExpectedSanitation["contextual_replacement.har"] = [][]string{
		{"cookieMonster1", contextualReplacement},
		{"cookieMonster1", contextualReplacement},
		{"cookieMonster1", contextualReplacement},
		{"cookieMonster2", contextualReplacement},
		{"cookieMonster2", contextualReplacement},
	}
	sanitizeFileToExpectedSanitation["remove_and_contextual_replacement.har"] = [][]string{
		{"cookieMonster1", contextualReplacement},
		{"cookieMonster2", contextualReplacement},
		{"cookieMonster1", contextualReplacement},
		{"7784689_72_76_104100_72_446760", "<REMOVED>"},
		{"7784689_72_76_104100_72_446760", "<REMOVED>"},
		{"7784689_72_76_104100_72_446760", "<REMOVED>"},
		{"7784689_72_76_104100_72_446760", "<REMOVED>"},
		{"7784689_72_76_104100_72_446760", "<REMOVED>"},
	}
	untouchedFileNames = []string{"1.har", "2.har", "3.har", "4.har", "5.har", "6.har", "7.har"}
}

func verifyFileSanitation(t *testing.T, fileDivElement selenium.WebElement) error {
	contextualReplacementMapping := make(map[string]string)

	fileNameElement, _ := fileDivElement.FindElement(selenium.ByCSSSelector, ".d2h-file-name")
	fileNameElementText, _ := fileNameElement.Text()
	fileNameElementFileNames := strings.Split(fileNameElementText, " → ")

	t.Log("filename = ", fileNameElementFileNames)
	originalFileName := fileNameElementFileNames[0]
	sanitizedFileName := fileNameElementFileNames[1]
	t.Log("Original filename = ", fileNameElementFileNames[0])
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
		t.Logf("Deleted Elements: %d\nInserted Elements: %d\n", len(deletedElements), len(insertedElements))
		deletedElementText, err := deletedElement.Text()

		if err != nil {
			return err
		}

		insertedElementText, err := insertedElements[index].Text()

		if err != nil {
			return err
		}
		expectedDeletedValue := expectedValues[index][0]
		expectedInsertedValue := expectedValues[index][1]

		t.Logf("Deleted Elements: %s\nInserted Elements: %s\n", deletedElementText, insertedElementText)
		assert.Equal(t, expectedDeletedValue, deletedElementText, "Did not delete expected parameter (%s), but removed {%s}", expectedDeletedValue, deletedElementText)

		if expectedInsertedValue == contextualReplacement {
			mappedInsertedValue, ok := contextualReplacementMapping[expectedDeletedValue]
			if !ok {
				contextualReplacementMapping[expectedDeletedValue] = insertedElementText
			} else {
				assert.Equal(t, mappedInsertedValue, insertedElementText, "Contextual Replacement shown different values. Expected %s, but got %s", mappedInsertedValue, insertedElementText)
			}
			expectedInsertedValue = insertedElementText

		} else {
			assert.Equal(t, expectedInsertedValue, insertedElementText, "Expected inserted value is %s, but got %s", expectedInsertedValue, insertedElementText)
		}

		lineNum1Element, err := lineNumElements[index].FindElement(selenium.ByCSSSelector, ".line-num1")
		if err != nil {
			return err
		}

		lineNum1Value, err := lineNum1Element.Text()

		if err != nil {
			return err
		}

		lineNum, err := strconv.Atoi(lineNum1Value)

		if err != nil {
			return fmt.Errorf("could not convert number %s: %s", lineNum1Value, err)
		}

		sanitizedFileChangeLines[lineNum] = expectedInsertedValue
	}
	return nil
}

func validE2EProcess(t *testing.T, webDriver selenium.WebDriver) error {
	var err error
	t.Log("Starting validating e2e process...\n")
	filesToSanitize := make([]string, len(sanitizeFileToExpectedSanitation)+len(untouchedFileNames))

	i := 0
	for fileToSanitize := range sanitizeFileToExpectedSanitation {
		filesToSanitize[i] = fileToSanitize
		i++
	}
	for index := range untouchedFileNames {
		filesToSanitize[i] = untouchedFileNames[index]
		i++
	}

	//Wait for page to be ready
	err = WaitForUploadButtonIsReady(webDriver, MaxWaitTimeout)
	if err != nil {
		return err
	}
	t.Log("Uploading File")
	err = UploadFiles(webDriver, filesToSanitize)
	if err != nil {
		return err
	}

	err = WaitForUntouchedFilesReady(webDriver, MaxWaitTimeout)

	if err != nil {
		return err
	}

	// Verify Untouched Files
	t.Logf("Verifying untouched files: %s", untouchedFileNames)
	unsanitizedElements, err := webDriver.FindElements(selenium.ByID, "unsanitized_file_p")

	if err != nil {
		return err
	}
	assert.Equal(t, len(untouchedFileNames), len(unsanitizedElements), "Expected untouched file names to be %d, but got %d", len(untouchedFileNames), unsanitizedElements)
	for _, unsanitizedElement := range unsanitizedElements {
		actualUnsanitizedFileName, err := unsanitizedElement.Text()
		if err != nil {
			return err
		}

		assert.True(t, slices.Contains(untouchedFileNames, actualUnsanitizedFileName), "Unexpected untouched file '%s' found. Expected untouched files: %s", actualUnsanitizedFileName, untouchedFileNames)
	}

	values, err := webDriver.FindElements(selenium.ByCSSSelector, ".d2h-file-wrapper")
	if err != nil {
		return err
	}
	fmt.Println("len of sanitized_diff_div", len(values))
	for _, fileDivElement := range values {
		err = verifyFileSanitation(t, fileDivElement)
		if err != nil {
			return err
		}
	}

	// Check download
	downloadButton, _ := webDriver.FindElement(selenium.ByID, "download_button_label")
	err = downloadButton.Click()
	if err != nil {
		return err
	}
	time.Sleep(25 * time.Second)

	for sanitizedFileName := range changedLines {
		expectedFileToDownloadPath := filepath.Join(DownloadPath, sanitizedFileName)
		_, downloadedFileError := os.Stat(expectedFileToDownloadPath)
		assert.Nil(t, downloadedFileError, downloadedFileError)
		err = verifyDownloadedFile(t, expectedFileToDownloadPath, changedLines[sanitizedFileName])
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

func verifyDownloadedFile(t *testing.T, filename string, changedLines map[int]string) error {

	inputFile, err := os.Open(filename)
	if err != nil {
		return err
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
	return nil
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

	err = WaitForNewWindowsOpen(webDriver, 2, MaxWaitTimeout)
	if err != nil {
		return err
	}

	handles, err := webDriver.WindowHandles()

	if err != nil {
		return err
	}

	// assert there's two windows
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

func invalidAlertDisplayedProcess(t *testing.T, webDriver selenium.WebDriver) error {
	filesToSanitize := []string{"invalid.har"}
	expectedAlertTextMessage := "Error parsing '" + filesToSanitize[0] + "'"
	return uploadAndVerifyAlerts(t, webDriver, filesToSanitize, expectedAlertTextMessage)
}

func duplicatedFilesToSanitizeProcess(t *testing.T, webDriver selenium.WebDriver) error {
	filesToSanitize := []string{"github.com.har", "duplicated/github.com.har"}
	expectedAlertTextMessage := "Multiple files with the same name (github.com.har) isn't supported. Please choose files with different names."
	return uploadAndVerifyAlerts(t, webDriver, filesToSanitize, expectedAlertTextMessage)

}

func fileUploadExceedsSizeLimitProcess(t *testing.T, webDriver selenium.WebDriver) error {
	filesToSanitize := []string{"big.har"}
	expectedAlertTextMessage := "Size of file big.har (53 MB) exceeds maximum supported file size of 50 MB."
	return uploadAndVerifyAlerts(t, webDriver, filesToSanitize, expectedAlertTextMessage)
}

func fileUploadNumberExceedsLimitProcess(t *testing.T, webDriver selenium.WebDriver) error {
	filesToSanitize := []string{"1.har", "2.har", "3.har", "4.har", "5.har", "6.har", "7.har", "8.har",
		"contextual_replacement.har", "github.com.har", "remove_and_contextual_replacement.har"}

	expectedAlertTextMessage := "Cannot sanitize more than 10 files at a time.\nSelect a lesser number of files."
	return uploadAndVerifyAlerts(t, webDriver, filesToSanitize, expectedAlertTextMessage)
}

func TestFirefoxDriverValidProcess(t *testing.T) {
	RunE2ETest(FIREFOX, t, validE2EProcess)
}

func TestChromeDriverValidProcess(t *testing.T) {
	RunE2ETest(CHROME, t, validE2EProcess)
}

func TestInvalidHarFileFireFoxDriver(t *testing.T) {
	RunE2ETest(FIREFOX, t, invalidAlertDisplayedProcess)
}

func TestInvalidHarFileChromeDriver(t *testing.T) {
	RunE2ETest(CHROME, t, invalidAlertDisplayedProcess)
}

func TestDuplicatedFileNamesFireFoxDriver(t *testing.T) {
	RunE2ETest(FIREFOX, t, duplicatedFilesToSanitizeProcess)
}

func TestDuplicatedFileNamesChromeDriver(t *testing.T) {
	RunE2ETest(CHROME, t, duplicatedFilesToSanitizeProcess)
}

func TestFileUploadExceedsSizeLimitFireFoxDriver(t *testing.T) {
	RunE2ETest(FIREFOX, t, fileUploadExceedsSizeLimitProcess)
}

func TestFileUploadExceedsSizeLimitChromeDriver(t *testing.T) {
	RunE2ETest(CHROME, t, fileUploadExceedsSizeLimitProcess)
}

func TestFileUploadNumberExceedsLimitFireFoxDriver(t *testing.T) {
	RunE2ETest(FIREFOX, t, fileUploadNumberExceedsLimitProcess)
}

func TestFileUploadNumberExceedsLimitChromeDriver(t *testing.T) {
	RunE2ETest(CHROME, t, fileUploadNumberExceedsLimitProcess)
}

func TestMain(m *testing.M) {
	SetUp()
	prepareMapTestScenario()
	m.Run()
	TearDown()
}

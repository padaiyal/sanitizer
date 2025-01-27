package e2e

import (
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"

	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tebeka/selenium"
)

var rulesFile = "rules/har.yaml"

func verifyFileSanitation(t *testing.T, fileDivElement selenium.WebElement, sanitizedFileInfos map[string]SanitizedFile) error {

	fileNameElement, _ := fileDivElement.FindElement(selenium.ByCSSSelector, ".d2h-file-name")
	fileNameElementText, _ := fileNameElement.Text()
	fileNameElementFileNames := strings.Split(fileNameElementText, " → ")

	t.Log("Original filename = ", fileNameElementFileNames[0])
	sanitizedFileInfo, ok := sanitizedFileInfos[fileNameElementFileNames[0]]
	if !ok {
		t.Fatal("SanitizedFileInfo not found")
	}
	assert.Equal(t, sanitizedFileInfo.sanitizedFilename, fileNameElementFileNames[1])

	deletedElements, err := fileDivElement.FindElements(selenium.ByCSSSelector, "td[class^='d2h-del d2h-change']")
	if err != nil {
		t.Fatal("Could not find deleted elements", err)
	}

	insertedElements, err := fileDivElement.FindElements(selenium.ByCSSSelector, "td[class^='d2h-ins d2h-change']")

	if err != nil {
		t.Fatal("Could not find inserted elements", err)
	}

	assert.Equal(t, len(deletedElements), len(insertedElements), "Deleted Elements length is different from inserted elements: %d vs. %d", len(deletedElements), len(insertedElements))

	t.Logf("Deleted Elements: %d\nInserted Elements: %d\n", len(deletedElements), len(insertedElements))
	for index, deletedElement := range deletedElements {

		deletedWithValueElement, err := deletedElement.FindElement(selenium.ByCSSSelector, ".d2h-code-line-ctn.hljs.plaintext")
		if err != nil {
			return err
		}

		deletedElementText, err := deletedWithValueElement.Text()

		insertedWithValueElement, err := insertedElements[index].FindElement(selenium.ByCSSSelector, ".d2h-code-line-ctn.hljs.plaintext")
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

func validE2EProcess(t *testing.T, webDriver selenium.WebDriver) error {
	var err error
	t.Log("Starting validating e2e process...\n")
	//fileNamesToSanitize := []string{"github.com.har", "contextual_replacement.har", "remove_and_contextual_replacement.har"}
	fileNamesToSanitize := []string{"github.com.har"}
	untouchedFileNames := []string{"1.har", "2.har", "3.har", "4.har", "5.har", "6.har", "7.har"}
	sanitizedFileInfos := make(map[string]SanitizedFile)

	filesToSanitizePath := append(fileNamesToSanitize, untouchedFileNames...)
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
	err = UploadFiles(webDriver, filesToSanitizePath)
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

	// Check download
	downloadButton, _ := webDriver.FindElement(selenium.ByID, "download_button_label")
	err = downloadButton.Click()
	if err != nil {
		return err
	}
	time.Sleep(25 * time.Second)

	for fileName, sanitizedFileInfo := range sanitizedFileInfos {

		expectedSanitizedFileNamePath := filepath.Join(ResourcesPath, "expected_sanitized_files", sanitizedFileInfo.sanitizedFilename)
		expectedSanitizedStatInfo, expectedSanitizedFileError := os.Stat(expectedSanitizedFileNamePath)
		assert.Nil(t, expectedSanitizedFileError, expectedSanitizedFileError)

		downloadedSanitizedFileNamePath := filepath.Join(DownloadPath, sanitizedFileInfo.sanitizedFilename)
		downloadedSanitizedStatInfo, downloadedSanitizedFileError := os.Stat(downloadedSanitizedFileNamePath)
		assert.Nil(t, downloadedSanitizedFileError, downloadedSanitizedFileError)

		assert.Equalf(t, expectedSanitizedStatInfo.Size(), downloadedSanitizedStatInfo.Size(), "expected sanitized file size '%d' is different from actual sanitized file size '%d'", expectedSanitizedStatInfo.Size(), downloadedSanitizedStatInfo.Size())

		matchingExpectedSanitizedFile, err := verifyDownloadedFileAndGetMatchingFileDiff(t, expectedSanitizedFileNamePath, downloadedSanitizedFileNamePath)

		if err != nil {
			return err
		}

		inputFilePath := filepath.Join(ResourcesPath, fileName)
		diffs, err := getExpectedUIDiffsFromMatchingSanitizedFileContent(t, inputFilePath, matchingExpectedSanitizedFile)
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
	fmt.Println("len of sanitized_diff_div", len(values))
	for _, fileDivElement := range values {
		err = verifyFileSanitation(t, fileDivElement, sanitizedFileInfos)
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

func getExpectedUIDiffsFromMatchingSanitizedFileContent(t *testing.T, inputSanitizedFileNamePath string, matchingExpectedSanitizedFileNamePath string) ([]diffmatchpatch.Diff, error) {
	inputSanitizedFileContentRaw, err := os.ReadFile(inputSanitizedFileNamePath)
	if err != nil {
		t.Errorf("Could not open file %s for reading: %s", inputSanitizedFileNamePath, err)
		return nil, err
	}
	inputSanitizedFileContent := string(inputSanitizedFileContentRaw)

	matchingExpectedSanitizedFileContentRaw, err := os.ReadFile(matchingExpectedSanitizedFileNamePath)
	if err != nil {
		t.Errorf("Could not open file %s for reading: %s", matchingExpectedSanitizedFileNamePath, err)
		return nil, err
	}
	matchingExpectedSanitizedFileContent := string(matchingExpectedSanitizedFileContentRaw)

	dmp := diffmatchpatch.New()

	inputSanitizedFileDmp, expectedSanitizedFileDmp, dmpStrings := dmp.DiffLinesToChars(inputSanitizedFileContent, matchingExpectedSanitizedFileContent)
	diffs := dmp.DiffMain(inputSanitizedFileDmp, expectedSanitizedFileDmp, false)
	diffs = dmp.DiffCharsToLines(diffs, dmpStrings)
	diffs2 := dmp.DiffCleanupSemantic(diffs)
	nonEqualDiffs := GetRemoveOrInsertInDiff(diffs2)
	//t.Logf("All Diffs From Original: %s\n", dmp.DiffPrettyText(diffs2))
	t.Logf("Diffs From Original: %s\n", dmp.DiffPrettyText(nonEqualDiffs))

	return nonEqualDiffs, nil

}

func verifyDownloadedFileAndGetMatchingFileDiff(t *testing.T, expectedSanitizedFileName string, actualSanitizedFileName string) (string, error) {
	var levenshteinDistance int
	dmp := diffmatchpatch.New()
	var diffs []diffmatchpatch.Diff
	actualSanitizedFileContentRaw, err := os.ReadFile(actualSanitizedFileName)
	if err != nil {
		t.Errorf("Could not open file %s for reading: %s", actualSanitizedFileName, err)
		return "", err
	}

	actualSanitizedFileContent := string(actualSanitizedFileContentRaw)
	expectedSanitizedFileContentRaw, err := os.ReadFile(expectedSanitizedFileName)
	if err != nil {
		t.Errorf("Could not open file %s for reading: %s", expectedSanitizedFileName, err)
		return "", err
	}
	expectedSanitizedFileContent := string(expectedSanitizedFileContentRaw)

	diffs = dmp.DiffMain(expectedSanitizedFileContent, actualSanitizedFileContent, true)
	levenshteinDistance = dmp.DiffLevenshtein(diffs)
	t.Logf("DiffLevenshtein distance of %d for expected file %s", dmp.DiffLevenshtein(diffs), expectedSanitizedFileName)

	assert.Equalf(t, 0, levenshteinDistance, "Expected sanitized file '%s' content matches actual sanitized file '%s' content.", expectedSanitizedFileName, actualSanitizedFileName)
	return expectedSanitizedFileName, nil
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

//func TestChromeDriverValidProcess(t *testing.T) {
//	RunE2ETest(CHROME, t, validE2EProcess)
//}
//
//func TestInvalidHarFileFireFoxDriver(t *testing.T) {
//	RunE2ETest(FIREFOX, t, invalidAlertDisplayedProcess)
//}
//
//func TestInvalidHarFileChromeDriver(t *testing.T) {
//	RunE2ETest(CHROME, t, invalidAlertDisplayedProcess)
//}
//
//func TestDuplicatedFileNamesFireFoxDriver(t *testing.T) {
//	RunE2ETest(FIREFOX, t, duplicatedFilesToSanitizeProcess)
//}
//
//func TestDuplicatedFileNamesChromeDriver(t *testing.T) {
//	RunE2ETest(CHROME, t, duplicatedFilesToSanitizeProcess)
//}
//
//func TestFileUploadExceedsSizeLimitFireFoxDriver(t *testing.T) {
//	RunE2ETest(FIREFOX, t, fileUploadExceedsSizeLimitProcess)
//}
//
//func TestFileUploadExceedsSizeLimitChromeDriver(t *testing.T) {
//	RunE2ETest(CHROME, t, fileUploadExceedsSizeLimitProcess)
//}
//
//func TestFileUploadNumberExceedsLimitFireFoxDriver(t *testing.T) {
//	RunE2ETest(FIREFOX, t, fileUploadNumberExceedsLimitProcess)
//}
//
//func TestFileUploadNumberExceedsLimitChromeDriver(t *testing.T) {
//	RunE2ETest(CHROME, t, fileUploadNumberExceedsLimitProcess)
//}

func TestMain(m *testing.M) {
	SetUp()
	m.Run()
	TearDown()
}

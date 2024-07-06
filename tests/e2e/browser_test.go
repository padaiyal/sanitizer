package e2e

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/tebeka/selenium"
)

type BrowserTestsSuite struct {
	suite.Suite
	Browser                   string
	service                   *selenium.Service
	driver                    selenium.WebDriver
	dynamicallyCreateHarPaths []string
	t                         *testing.T
}

func (suite *BrowserTestsSuite) SetupSuite() {
	suite.t = suite.T()

	suite.t.Logf("Running tests for browser %s", suite.Browser)
	har1Path := GetInputFilePath("1.har")
	suite.dynamicallyCreateHarPaths = []string{}
	for i := 2; i <= 8; i += 1 {
		suite.dynamicallyCreateHarPaths = append(suite.dynamicallyCreateHarPaths, GetInputFilePath(fmt.Sprintf("%d.har", i)))
	}

	err := CopyFile(har1Path, suite.dynamicallyCreateHarPaths)
	if err != nil {
		suite.t.Fatalf("Could not copy %s to %s. Error:%s\n", har1Path, suite.dynamicallyCreateHarPaths, err)
	}
}

func (suite *BrowserTestsSuite) TearDownSuite() {
	for _, path := range suite.dynamicallyCreateHarPaths {
		_, err := os.Stat(path)
		if err == nil {
			err = os.Remove(path)
			if err != nil {
				suite.t.Logf("Could not delete file (%s) due to %s", path, err)
			}
		} else if !os.IsNotExist(err) {
			suite.t.Logf("Could not delete file (%s) due to %s", path, err)
		}
	}
}

func (suite *BrowserTestsSuite) SetupTest() {
	var err error
	suite.t.Logf("Setting up test for browser: %s", suite.Browser)
	ResetEnvironment()
	suite.service, suite.driver, err = GetDriver(suite.Browser)
	if err != nil {
		suite.t.Fatalf("Error getting driver: %s", err)
	}

}

func (suite *BrowserTestsSuite) TearDownTest() {
	CloseWebDriverAndService(suite.driver, suite.service)
}

func (suite *BrowserTestsSuite) TestValidFlow() {
	var err error
	t := suite.T()

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
	err = WaitForUploadButtonIsReady(suite.driver, MaxWaitTimeout)
	if err != nil {
		suite.t.Fatalf("Error running test: %s", err)
	}

	t.Log("Uploading File")
	filesToSanitizePath := append(fileNamesToSanitize, untouchedFileNames...)
	err = UploadFiles(suite.driver, filesToSanitizePath)
	if err != nil {
		suite.t.Fatalf("Error running test: %s", err)
	}

	// Wait until the page is ready. Untouched file element was chosen arbitrarily
	err = WaitForUntouchedFilesReady(suite.driver, MaxWaitTimeout)
	if err != nil {
		suite.t.Fatalf("Error running test: %s", err)
	}

	// Verify Untouched Files
	t.Logf("Verifying untouched files: %s", untouchedFileNames)
	unSanitizedElements, err := suite.driver.FindElements(selenium.ByID, "unsanitized_file_p")
	if err != nil {
		suite.t.Fatalf("Error running test: %s", err)
	}

	assert.Equal(t, len(untouchedFileNames), len(unSanitizedElements),
		"Expected untouched file names to be %d, but got %d", len(untouchedFileNames), unSanitizedElements)
	for _, unSanitizedElement := range unSanitizedElements {
		actualUnSanitizedFileName, err := unSanitizedElement.Text()
		if err != nil {
			suite.t.Fatalf("Error running test: %s", err)
		}

		assert.True(t, slices.Contains(untouchedFileNames, actualUnSanitizedFileName),
			"Unexpected untouched file '%s' found. Expected untouched files: %s", actualUnSanitizedFileName, untouchedFileNames)
	}

	// Verify downloads
	downloadButton, _ := suite.driver.FindElement(selenium.ByID, "download_button_label")
	err = downloadButton.Click()
	if err != nil {
		suite.t.Fatalf("Error running test: %s", err)
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
		inputFilePath := GetInputFilePath(fileName)
		diffs, err := getExpectedUIDiffsFromMatchingSanitizedFileContent(t, inputFilePath, downloadedSanitizedFileNamePath)
		if err != nil {
			suite.t.Fatalf("Error running test: %s", err)
		}
		sanitizedFileInfo.lines = CreateMappingOfDeletedAndInsertedStringsInDiff(diffs)
		sanitizedFileInfos[fileName] = sanitizedFileInfo

	}

	values, err := suite.driver.FindElements(selenium.ByCSSSelector, ".d2h-file-wrapper")
	if err != nil {
		suite.t.Fatalf("Error running test: %s", err)
	}

	for _, fileDivElement := range values {
		err = verifyFileSanitizationInUI(t, fileDivElement, sanitizedFileInfos)
		if err != nil {
			suite.t.Fatalf("Error running test: %s", err)
		}
	}

	// View Rules
	err = verifyViewRules(t, suite.driver)
	if err != nil {
		suite.t.Fatalf("Error running test: %s", err)
	}

}

func (suite *BrowserTestsSuite) TestInvalidHarFile() {
	filesToSanitize := []string{"invalid.har"}
	expectedAlertTextMessage := "Error parsing '" + filesToSanitize[0] + "'"
	uploadAndVerifyAlerts(suite.t, suite.driver, filesToSanitize, expectedAlertTextMessage)
}

func (suite *BrowserTestsSuite) TestFileUploadExceedsSizeLimit() {
	filesToSanitize := []string{"big.har"}
	expectedAlertTextMessage := "Size of file big.har (53 MB) exceeds maximum supported file size of 50 MB."
	uploadAndVerifyAlerts(suite.t, suite.driver, filesToSanitize, expectedAlertTextMessage)
}

func (suite *BrowserTestsSuite) TestDuplicatedFileNames() {

	filesToSanitize := []string{"github.com.har", "duplicated/github.com.har"}
	expectedAlertTextMessage := "Multiple files with the same name (github.com.har) isn't supported. Please choose files with different names."
	uploadAndVerifyAlerts(suite.t, suite.driver, filesToSanitize, expectedAlertTextMessage)

}

func (suite *BrowserTestsSuite) TestFileUploadNumberExceedsLimit() {
	filesToSanitize := []string{"1.har", "2.har", "3.har", "4.har", "5.har", "6.har", "7.har", "8.har",
		"contextual_replacement.har", "github.com.har", "remove_and_contextual_replacement.har"}

	expectedAlertTextMessage := "Cannot sanitize more than 10 files at a time.\nSelect a lesser number of files."
	uploadAndVerifyAlerts(suite.t, suite.driver, filesToSanitize, expectedAlertTextMessage)
}

func TestBrowserTestSuite(t *testing.T) {
	suite.Run(t, &BrowserTestsSuite{Browser: FIREFOX})
	suite.Run(t, &BrowserTestsSuite{Browser: CHROME})
}

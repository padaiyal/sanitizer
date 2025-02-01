package e2e

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type ChromeTestSuite struct {
	suite.Suite
	Browser string
}

func (suite *ChromeTestSuite) SetupSuite() {
	suite.Browser = CHROME
}

func (suite *ChromeTestSuite) TearDownSuite() {
	ResetEnvironment()
}

func (suite *ChromeTestSuite) TearDownTest() {
	//CloseWebDriverAndService(suite.driver, suite.service)
}

func (suite *ChromeTestSuite) TestChromeDriverValidProcess() {
	RunE2ETestParallel(suite.T(), suite.Browser, true, TestingValidE2E)
}

func (suite *ChromeTestSuite) TestInvalidHarFileChromeDriver() {
	RunE2ETestParallel(suite.T(), suite.Browser, false, TestingInvalidAlertDisplayed)
}

func (suite *ChromeTestSuite) TestFileUploadExceedsSizeLimitChromeDriver() {
	RunE2ETestParallel(suite.T(), suite.Browser, false, TestingFileUploadExceedsSizeLimit)
}

func (suite *ChromeTestSuite) TestDuplicatedFileNamesChromeDriver() {
	RunE2ETestParallel(suite.T(), suite.Browser, false, TestingDuplicatedFilesToSanitize)
}

func (suite *ChromeTestSuite) TestFileUploadNumberExceedsLimitChromeDriver() {
	RunE2ETestParallel(suite.T(), suite.Browser, false, TestingFileUploadNumberExceedsLimit)
}

func TestChrome(t *testing.T) {
	suite.Run(t, new(ChromeTestSuite))
}

package e2e

import (
	"github.com/stretchr/testify/suite"
	"github.com/tebeka/selenium"
	"log"
	"testing"
)

type ChromeTestSuite struct {
	suite.Suite
	Browser string
	service *selenium.Service
	driver  selenium.WebDriver
}

func (suite *ChromeTestSuite) SetupSuite() {
	suite.Browser = CHROME
}

func (suite *ChromeTestSuite) TearDownSuite() {
}

func (suite *ChromeTestSuite) SetupTest() {

	var err error

	ResetEnvironment()
	suite.service, suite.driver, err = GetDriver(suite.Browser)
	if err != nil {
		log.Fatalf("Error getting driver: %s", err)
	}

}

func (suite *ChromeTestSuite) TearDownTest() {
	CloseWebDriverAndService(suite.driver, suite.service)
}

func (suite *ChromeTestSuite) TestChromeDriverValidProcess() {
	err := TestingValidE2E(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
}

func (suite *ChromeTestSuite) TestInvalidHarFileChromeDriver() {
	err := TestingInvalidAlertDisplayed(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
}

func (suite *ChromeTestSuite) TestFileUploadExceedsSizeLimitChromeDriver() {
	err := TestingFileUploadExceedsSizeLimit(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
}

func (suite *ChromeTestSuite) TestDuplicatedFileNamesChromeDriver() {
	err := TestingDuplicatedFilesToSanitize(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
}

func (suite *ChromeTestSuite) TestFileUploadNumberExceedsLimitChromeDriver() {
	err := TestingFileUploadNumberExceedsLimit(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
}

func TestBrowserTestSuite(t *testing.T) {
	suite.Run(t, new(ChromeTestSuite))
}

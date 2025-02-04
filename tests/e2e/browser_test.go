package e2e

import (
	"log"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tebeka/selenium"
)

type BrowserTestSuite struct {
	suite.Suite
	Browser string
	service *selenium.Service
	driver  selenium.WebDriver
}

func (suite *BrowserTestSuite) SetupSuite() {
	log.Print("Running browser tests for " + suite.Browser)
}

func (suite *BrowserTestSuite) SetupTest() {

	var err error

	ResetEnvironment()
	suite.service, suite.driver, err = GetDriver(suite.Browser)
	if err != nil {
		log.Fatalf("Error getting driver: %s", err)
	}

}

func (suite *BrowserTestSuite) TearDownTest() {
	CloseWebDriverAndService(suite.driver, suite.service)
}

func (suite *BrowserTestSuite) TestValidFlow() {
	err := TestingValidE2E(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
}

func (suite *BrowserTestSuite) TestInvalidHarFile() {
	err := TestingInvalidAlertDisplayed(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
}

func (suite *BrowserTestSuite) TestFileUploadExceedsSizeLimit() {
	err := TestingFileUploadExceedsSizeLimit(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
}

func (suite *BrowserTestSuite) TestDuplicatedFileNames() {
	err := TestingDuplicatedFilesToSanitize(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
}

func (suite *BrowserTestSuite) TestFileUploadNumberExceedsLimit() {
	err := TestingFileUploadNumberExceedsLimit(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
}

func TestBrowserTestSuite(t *testing.T) {
	suite.Run(t, &BrowserTestSuite{Browser: FIREFOX})
	suite.Run(t, &BrowserTestSuite{Browser: CHROME})
}

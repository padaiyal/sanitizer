package e2e

import (
	"github.com/stretchr/testify/suite"
	"github.com/tebeka/selenium"
	"log"
	"testing"
)

type FirefoxTestSuite struct {
	suite.Suite
	Browser string
	service *selenium.Service
	driver  selenium.WebDriver
}

func (suite *FirefoxTestSuite) SetupSuite() {
	suite.Browser = FIREFOX
}

func (suite *FirefoxTestSuite) SetupTest() {

	var err error

	ResetEnvironment()
	suite.service, suite.driver, err = GetDriver(suite.Browser)
	if err != nil {
		log.Fatalf("Error getting driver: %s", err)
	}

}

func (suite *FirefoxTestSuite) TearDownTest() {
	CloseWebDriverAndService(suite.driver, suite.service)
}

func (suite *FirefoxTestSuite) TestFirefoxDriverValidProcess() {
	err := TestingValidE2E(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
	suite.T().Log("Finished running `TestFirefoxDriverValidProcess'")
}

func (suite *FirefoxTestSuite) TestInvalidHarFileFirefoxDriver() {
	err := TestingInvalidAlertDisplayed(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
	suite.T().Log("Finished running `TestInvalidHarFileFirefoxDriver'")
}

func (suite *FirefoxTestSuite) TestFileUploadExceedsSizeLimitFirefoxDriver() {
	err := TestingFileUploadExceedsSizeLimit(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
}

func (suite *FirefoxTestSuite) TestDuplicatedFileNamesFirefoxDriver() {
	err := TestingDuplicatedFilesToSanitize(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
	suite.T().Log("Finished running `TestDuplicatedFileNamesFirefoxDriver'")
}

func (suite *FirefoxTestSuite) TestFileUploadNumberExceedsLimitFirefoxDriver() {
	err := TestingFileUploadNumberExceedsLimit(suite.T(), suite.driver)
	if err != nil {
		log.Fatalf("Error running test: %s", err)
	}
	suite.T().Log("Finished running `TestFileUploadNumberExceedsLimitFirefoxDriver'")
}

func TestFirefoxFirefoxTestSuite(t *testing.T) {
	suite.Run(t, new(FirefoxTestSuite))
}

package e2e

//
//import (
//	"github.com/stretchr/testify/suite"
//	"testing"
//)
//
//type FirefoxTestSuite struct {
//	suite.Suite
//	Browser string
//}
//
//func (suite *FirefoxTestSuite) SetupSuite() {
//	suite.Browser = FIREFOX
//}
//
//func (suite *FirefoxTestSuite) TearDownSuite() {
//	ResetEnvironment()
//}
//
//func (suite *FirefoxTestSuite) TearDownTest() {
//	//CloseWebDriverAndService(suite.driver, suite.service)
//}
//
//func (suite *FirefoxTestSuite) TestFirefoxDriverValidProcess() {
//	RunE2ETestParallel(suite.T(), suite.Browser, true, TestingValidE2E)
//}
//
//func (suite *FirefoxTestSuite) TestInvalidHarFileFirefoxDriver() {
//	RunE2ETestParallel(suite.T(), suite.Browser, false, TestingInvalidAlertDisplayed)
//}
//
//func (suite *FirefoxTestSuite) TestFileUploadExceedsSizeLimitFirefoxDriver() {
//	RunE2ETestParallel(suite.T(), suite.Browser, false, TestingFileUploadExceedsSizeLimit)
//}
//
//func (suite *FirefoxTestSuite) TestDuplicatedFileNamesFirefoxDriver() {
//	RunE2ETestParallel(suite.T(), suite.Browser, false, TestingDuplicatedFilesToSanitize)
//}
//
//func (suite *FirefoxTestSuite) TestFileUploadNumberExceedsLimitFirefoxDriver() {
//	RunE2ETestParallel(suite.T(), suite.Browser, false, TestingFileUploadNumberExceedsLimit)
//}
//
//func TestFirefox(t *testing.T) {
//	suite.Run(t, new(FirefoxTestSuite))
//}

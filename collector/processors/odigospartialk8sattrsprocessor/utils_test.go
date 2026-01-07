package odigospartialk8sattrsprocessor

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// UtilsTestSuite tests the utility functions
type UtilsTestSuite struct {
	suite.Suite
}

func TestUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(UtilsTestSuite))
}

func (s *UtilsTestSuite) TestExtractServiceNameWithSuffix_DeploymentWithReplicasetSuffix() {
	result, err := extractServiceNameWithSuffix("my-service-5d4b7c8f9")
	s.Require().NoError(err)
	s.Equal("my-service", result)
}

func (s *UtilsTestSuite) TestExtractServiceNameWithSuffix_MultipleHyphens() {
	result, err := extractServiceNameWithSuffix("my-awesome-service-5d4b7c8f9")
	s.Require().NoError(err)
	s.Equal("my-awesome-service", result)
}

func (s *UtilsTestSuite) TestExtractServiceNameWithSuffix_SingleHyphen() {
	result, err := extractServiceNameWithSuffix("service-abc123")
	s.Require().NoError(err)
	s.Equal("service", result)
}

func (s *UtilsTestSuite) TestExtractServiceNameWithSuffix_ComplexName() {
	result, err := extractServiceNameWithSuffix("frontend-api-v2-deployment-abc123def456")
	s.Require().NoError(err)
	s.Equal("frontend-api-v2-deployment", result)
}

func (s *UtilsTestSuite) TestExtractServiceNameWithSuffix_NoHyphen() {
	result, err := extractServiceNameWithSuffix("servicename")
	s.Require().Error(err)
	s.Contains(err.Error(), "does not contain a hyphen")
	s.Equal("", result)
}

func (s *UtilsTestSuite) TestExtractServiceNameWithSuffix_EmptyString() {
	result, err := extractServiceNameWithSuffix("")
	s.Require().Error(err)
	s.Contains(err.Error(), "does not contain a hyphen")
	s.Equal("", result)
}

func (s *UtilsTestSuite) TestExtractServiceNameWithSuffix_HyphenAtStart() {
	result, err := extractServiceNameWithSuffix("-suffix")
	s.Require().NoError(err)
	s.Equal("", result)
}

func (s *UtilsTestSuite) TestExtractServiceNameWithSuffix_HyphenAtEnd() {
	result, err := extractServiceNameWithSuffix("prefix-")
	s.Require().NoError(err)
	s.Equal("prefix", result)
}

func (s *UtilsTestSuite) TestExtractServiceNameWithSuffix_OnlyHyphen() {
	result, err := extractServiceNameWithSuffix("-")
	s.Require().NoError(err)
	s.Equal("", result)
}

func (s *UtilsTestSuite) TestExtractServiceNameWithSuffix_DaemonsetStyle() {
	result, err := extractServiceNameWithSuffix("odiglet-xk7np")
	s.Require().NoError(err)
	s.Equal("odiglet", result)
}

func (s *UtilsTestSuite) TestExtractServiceNameWithSuffix_StatefulsetStyle() {
	result, err := extractServiceNameWithSuffix("postgres-0")
	s.Require().NoError(err)
	s.Equal("postgres", result)
}

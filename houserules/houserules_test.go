package houserules_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/coalaura/vet/houserules"
)

func TestHouseRules(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), houserules.Analyzer, "rules")
}

func TestFunctionParameterFixes(t *testing.T) {
	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), houserules.Analyzer, "parameters")
}

func TestBreathe(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), houserules.Breathe, "breathe")
}

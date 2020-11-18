// +build legacy

package multierr

import (
	"math/rand"
	"strings"
	"testing"
)

var testErrors []*multiError
var sampleText = strings.Repeat("sampletext", 10)

type testErr string

func (e testErr) Error() string { return string(e) }

const (
	nErrors = 2 << 10
	seed    = 8012
)

func init() {
	var rnd = rand.New(rand.NewSource(seed))
	var maxText = len(sampleText)
	var row []error
	for i := 0; i < nErrors; i++ {
		row = row[:0]
		var toGenerate = rnd.Intn(3) + 2
		for j := 0; j < toGenerate; j++ {
			var n = rnd.Intn(maxText) + 1
			var text = sampleText[:n]
			row = append(row, testErr(text))
		}
		testErrors = append(testErrors, Combine(row...).(*multiError))
	}
}

var dump string

func BenchmarkError(bench *testing.B) {
	bench.ReportAllocs()
	for i := 0; i < bench.N; i++ {
		for _, err := range testErrors {
			dump = err.Error()
		}
	}
}

func BenchmarkErrorLegacy(bench *testing.B) {
	bench.ReportAllocs()
	for i := 0; i < bench.N; i++ {
		for _, err := range testErrors {
			dump = err.oldError()
		}
	}
}

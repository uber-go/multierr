package multierr_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
)

func TestCombineFlattensErrorsJoin(t *testing.T) {
	a := errors.New("a")
	b := errors.New("b")
	c := errors.New("c")
	joined := errors.Join(a, b)
	got := multierr.Combine(joined, c)
	require.Error(t, got)
	errs := multierr.Errors(got)
	// Should be flat: a, b, c — not [join(a,b), c]
	assert.Len(t, errs, 3)
	assert.True(t, errors.Is(got, a))
	assert.True(t, errors.Is(got, b))
	assert.True(t, errors.Is(got, c))
}

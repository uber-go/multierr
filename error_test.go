package multierr

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// richFormatError is an error that prints a different output depending on
// whether %v or %+v was used.
type richFormatError struct{}

func (r richFormatError) Error() string {
	return fmt.Sprint(r)
}

func (richFormatError) Format(f fmt.State, c rune) {
	if c == 'v' && f.Flag('+') {
		io.WriteString(f, "multiline\nmessage\nwith plus")
	} else {
		io.WriteString(f, "without plus")
	}
}

func appendN(initial, err error, n int) error {
	errs := initial
	for i := 0; i < n; i++ {
		errs = Append(errs, err)
	}
	return errs
}

func TestCombine(t *testing.T) {
	tests := []struct {
		giveErrors     []error
		wantError      error
		wantMultiline  string
		wantSingleline string
	}{
		{
			giveErrors: nil,
			wantError:  nil,
		},
		{
			giveErrors: []error{},
			wantError:  nil,
		},
		{
			giveErrors: []error{
				errors.New("foo"),
				nil,
				multiError{
					errors.New("bar"),
				},
				nil,
			},
			wantError: multiError{
				errors.New("foo"),
				errors.New("bar"),
			},
			wantMultiline: "the following errors occurred:\n" +
				" -  foo\n" +
				" -  bar",
			wantSingleline: "foo; bar",
		},
		{
			giveErrors: []error{
				errors.New("foo"),
				multiError{
					errors.New("bar"),
				},
			},
			wantError: multiError{
				errors.New("foo"),
				errors.New("bar"),
			},
			wantMultiline: "the following errors occurred:\n" +
				" -  foo\n" +
				" -  bar",
			wantSingleline: "foo; bar",
		},
		{
			giveErrors:     []error{errors.New("great sadness")},
			wantError:      errors.New("great sadness"),
			wantMultiline:  "great sadness",
			wantSingleline: "great sadness",
		},
		{
			giveErrors: []error{
				errors.New("foo"),
				errors.New("bar"),
			},
			wantError: multiError{
				errors.New("foo"),
				errors.New("bar"),
			},
			wantMultiline: "the following errors occurred:\n" +
				" -  foo\n" +
				" -  bar",
			wantSingleline: "foo; bar",
		},
		{
			giveErrors: []error{
				errors.New("great sadness"),
				errors.New("multi\n  line\nerror message"),
				errors.New("single line error message"),
			},
			wantError: multiError{
				errors.New("great sadness"),
				errors.New("multi\n  line\nerror message"),
				errors.New("single line error message"),
			},
			wantMultiline: "the following errors occurred:\n" +
				" -  great sadness\n" +
				" -  multi\n" +
				"      line\n" +
				"    error message\n" +
				" -  single line error message",
			wantSingleline: "great sadness; " +
				"multi\n  line\nerror message; " +
				"single line error message",
		},
		{
			giveErrors: []error{
				errors.New("foo"),
				multiError{
					errors.New("bar"),
					errors.New("baz"),
				},
				errors.New("qux"),
			},
			wantError: multiError{
				errors.New("foo"),
				errors.New("bar"),
				errors.New("baz"),
				errors.New("qux"),
			},
			wantMultiline: "the following errors occurred:\n" +
				" -  foo\n" +
				" -  bar\n" +
				" -  baz\n" +
				" -  qux",
			wantSingleline: "foo; bar; baz; qux",
		},
		{
			giveErrors: []error{
				errors.New("foo"),
				nil,
				multiError{
					errors.New("bar"),
				},
				nil,
			},
			wantError: multiError{
				errors.New("foo"),
				errors.New("bar"),
			},
			wantMultiline: "the following errors occurred:\n" +
				" -  foo\n" +
				" -  bar",
			wantSingleline: "foo; bar",
		},
		{
			giveErrors: []error{
				errors.New("foo"),
				multiError{
					errors.New("bar"),
				},
			},
			wantError: multiError{
				errors.New("foo"),
				errors.New("bar"),
			},
			wantMultiline: "the following errors occurred:\n" +
				" -  foo\n" +
				" -  bar",
			wantSingleline: "foo; bar",
		},
		{
			giveErrors: []error{
				errors.New("foo"),
				richFormatError{},
				errors.New("bar"),
			},
			wantError: multiError{
				errors.New("foo"),
				richFormatError{},
				errors.New("bar"),
			},
			wantMultiline: "the following errors occurred:\n" +
				" -  foo\n" +
				" -  multiline\n" +
				"    message\n" +
				"    with plus\n" +
				" -  bar",
			wantSingleline: "foo; without plus; bar",
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			err := Combine(tt.giveErrors...)
			require.Equal(t, tt.wantError, err)

			if tt.wantMultiline != "" {
				assert.Equal(t, tt.wantMultiline, fmt.Sprintf("%+v", err))
			}

			if tt.wantSingleline != "" {
				assert.Equal(t, tt.wantSingleline, err.Error())
				if s, ok := err.(fmt.Stringer); ok {
					assert.Equal(t, tt.wantSingleline, s.String())
				}
				assert.Equal(t, tt.wantSingleline, fmt.Sprintf("%v", err))
			}
		})
	}
}

func TestCombineDoesNotModifySlice(t *testing.T) {
	errors := []error{
		errors.New("foo"),
		nil,
		errors.New("bar"),
	}

	assert.NotNil(t, Combine(errors...))
	assert.Len(t, errors, 3)
	assert.Nil(t, errors[1], 3)
}

func TestAppend(t *testing.T) {
	tests := []struct {
		left  error
		right error
		want  error
	}{
		{
			left:  nil,
			right: nil,
			want:  nil,
		},
		{
			left:  nil,
			right: errors.New("great sadness"),
			want:  errors.New("great sadness"),
		},
		{
			left:  errors.New("great sadness"),
			right: nil,
			want:  errors.New("great sadness"),
		},
		{
			left:  errors.New("foo"),
			right: errors.New("bar"),
			want: multiError{
				errors.New("foo"),
				errors.New("bar"),
			},
		},
		{
			left: multiError{
				errors.New("foo"),
				errors.New("bar"),
			},
			right: errors.New("baz"),
			want: multiError{
				errors.New("foo"),
				errors.New("bar"),
				errors.New("baz"),
			},
		},
		{
			left: errors.New("baz"),
			right: multiError{
				errors.New("foo"),
				errors.New("bar"),
			},
			want: multiError{
				errors.New("baz"),
				errors.New("foo"),
				errors.New("bar"),
			},
		},
		{
			left: multiError{
				errors.New("foo"),
			},
			right: multiError{
				errors.New("bar"),
			},
			want: multiError{
				errors.New("foo"),
				errors.New("bar"),
			},
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, Append(tt.left, tt.right))
	}
}

func TestAppendDoesNotModify(t *testing.T) {
	createInitial := func() error {
		// Create a multiError that has capacity for more errors so Append will
		// modify the underlying array that may be shared.
		return appendN(nil, errors.New("initial"), 50)
	}

	initial := createInitial()
	err1 := Append(initial, errors.New("err1"))
	err2 := Append(initial, errors.New("err2"))

	// initial should not have been modified.
	assert.Equal(t, createInitial(), initial, "Initial should not be modified")

	assert.Equal(t, Append(createInitial(), errors.New("err1")), err1)
	assert.Equal(t, Append(createInitial(), errors.New("err2")), err2)
}

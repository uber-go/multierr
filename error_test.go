package multierr

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type myErrorGroup []error

func (e myErrorGroup) Error() string {
	messages := make([]string, len(e))
	for i, err := range e {
		messages[i] = err.Error()
	}
	return strings.Join(messages, "\n")
}

func (e myErrorGroup) Causes() []error {
	return e
}

func TestFromSlice(t *testing.T) {
	tests := []struct {
		giveErrors  []error
		wantError   error
		wantMessage string
	}{
		{
			giveErrors: []error{},
			wantError:  nil,
		},
		{
			giveErrors:  []error{errors.New("great sadness")},
			wantError:   errors.New("great sadness"),
			wantMessage: "great sadness",
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
			wantMessage: "the following errors occurred:\n" +
				" -  foo\n" +
				" -  bar",
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
			wantMessage: "the following errors occurred:\n" +
				" -  great sadness\n" +
				" -  multi\n" +
				"      line\n" +
				"    error message\n" +
				" -  single line error message",
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
			wantMessage: "the following errors occurred:\n" +
				" -  foo\n" +
				" -  bar\n" +
				" -  baz\n" +
				" -  qux",
		},
		{
			giveErrors: []error{
				errors.New("foo"),
				myErrorGroup{
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
			wantMessage: "the following errors occurred:\n" +
				" -  foo\n" +
				" -  bar\n" +
				" -  baz\n" +
				" -  qux",
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			err := FromSlice(tt.giveErrors)
			if assert.Equal(t, tt.wantError, err) && tt.wantMessage != "" {
				assert.Equal(t, tt.wantMessage, err.Error())
			}
		})
	}
}

func TestCombine(t *testing.T) {
	tests := []struct {
		give []error
		want error
	}{
		{
			give: []error{
				errors.New("foo"),
				nil,
				multiError{
					errors.New("bar"),
				},
				nil,
			},
			want: multiError{
				errors.New("foo"),
				errors.New("bar"),
			},
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, Combine(tt.give...))
	}
}

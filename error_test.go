package multierr

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestFromSlice(t *testing.T) {
	tests := []struct {
		giveErrors     []error
		wantError      error
		wantMultiline  string
		wantSingleline string
	}{
		{
			giveErrors: []error{},
			wantError:  nil,
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
	}

	for i, tt := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			err := FromSlice(tt.giveErrors)
			if !reflect.DeepEqual(tt.wantError, err) {
				t.Fatalf("FromSlice output mismatch:\n\twant: %#v\n\t got: %#v", tt.wantError, err)
			}

			if tt.wantMultiline != "" {
				if got := fmt.Sprintf("%+v", err); tt.wantMultiline != got {
					t.Errorf("%%+v output did not match:\n\twant: %q\n\t got: %q", tt.wantMultiline, got)
				}
			}

			if tt.wantSingleline != "" {
				if got := err.Error(); tt.wantSingleline != got {
					t.Errorf("Error() output did not match:\n\twant: %q\n\t got: %q", tt.wantSingleline, got)
				}

				if s, ok := err.(fmt.Stringer); ok {
					if got := s.String(); tt.wantSingleline != got {
						t.Errorf("String() output did not match:\n\twant: %q\n\t got: %q", tt.wantSingleline, got)
					}
				}

				if got := fmt.Sprintf("%v", err); tt.wantSingleline != got {
					t.Errorf("%%v output did not match:\n\twant: %q\n\t got: %q", tt.wantSingleline, got)
				}
			}
		})
	}
}

func TestFromSliceDoesNotModifySlice(t *testing.T) {
	errors := []error{
		errors.New("foo"),
		nil,
		errors.New("bar"),
	}

	if FromSlice(errors) == nil {
		t.Errorf("expected non-nil output from FromSlice")
	}

	if len(errors) != 3 {
		t.Errorf("length of errors was changed by FromSlice:\n\twant: 3\n\t got: %d", len(errors))
	}

	if errors[1] != nil {
		t.Errorf("expected errors[1] to be nil but got %v", errors[1])
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
		{
			give: []error{
				errors.New("foo"),
				multiError{
					errors.New("bar"),
				},
			},
			want: multiError{
				errors.New("foo"),
				errors.New("bar"),
			},
		},
	}

	for _, tt := range tests {
		if err := Combine(tt.give...); !reflect.DeepEqual(tt.want, err) {
			t.Errorf("Combine output mismatch:\n\twant: %#v\n\t got: %#v", tt.want, err)
		}
	}
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
		if err := Append(tt.left, tt.right); !reflect.DeepEqual(tt.want, err) {
			t.Errorf("Append output mismatch:\n\twant: %#v\n\t got: %#v", tt.want, err)
		}
	}
}

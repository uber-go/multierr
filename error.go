// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package multierr

import "strings"

// DefaultPrefix is the prefix used for combined error messages. To use a
// different prefix, pass the Prefix option to FromSlice.
const DefaultPrefix = "The following errors occurred:"

// Amount of space we reserve in a slice when flattening nested errorGroups.
const _errorBuffer = 8

// errorGroup is an interface implemented by any error type which combines one
// or more errors.
type errorGroup interface {
	Causes() []error
}

type multiError struct {
	Prefix string
	Errors []error
}

func (me *multiError) String() string {
	return me.Error()
}

func (me *multiError) Causes() []error {
	return me.Errors
}

func (me *multiError) Error() string {
	msg := me.Prefix
	if msg == "" {
		msg = DefaultPrefix
	}
	for _, err := range me.Errors {
		msg += "\n -  " + indentTail(4, err.Error())
	}
	return msg
}

// indentTail prepends the given number of spaces to all lines following the
// first line of the given string.
func indentTail(spaces int, s string) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i, line := range lines[1:] {
		lines[i+1] = prefix + line
	}
	return strings.Join(lines, "\n")
}

// Option customizes a multierr error.
type Option struct{ apply func(*multiError) }

// Prefix changes the prefix that will be printed before the list of error
// messages. Defaults to DefaultPrefix.
func Prefix(prefix string) Option {
	return Option{apply: func(me *multiError) {
		me.Prefix = prefix
	}}
}

// flatten flattens nested errorGroups into a single list of errors.
func flatten(errors []error) []error {
	errors = filterNil(errors)
	if len(errors) == 0 {
		return nil
	}

	// zero-alloc path: no nested errors
	idx := findFirstErrorGroup(errors)
	if idx < 0 {
		return errors
	}

	// If an errorGroup was found, we need to write to a new list
	newErrors := make([]error, 0, len(errors)+_errorBuffer)
	newErrors = append(newErrors, errors[:idx]...)
	return flattenTo(newErrors, errors[idx:])
}

// filterNil removes nil errors from the given slice.
func filterNil(errors []error) []error {
	// zero-alloc filtering
	newErrors := errors[:0]
	for _, err := range errors {
		if err != nil {
			newErrors = append(newErrors, err)
		}
	}
	return newErrors
}

// Finds the first index in the given slice where an errorGroup was found
// instead of an error.
//
// -1 is returned if none of the errors were an errorGroup.
func findFirstErrorGroup(errors []error) int {
	for i, err := range errors {
		if _, ok := err.(errorGroup); ok {
			return i
		}
	}
	return -1
}

// flattenTo flattens the src list of errors by appending to dest. Returns the
// final dest list.
func flattenTo(dest, src []error) []error {
	for _, err := range src {
		if err == nil {
			continue
		}
		if e, ok := err.(errorGroup); ok {
			dest = flattenTo(dest, e.Causes())
		} else {
			dest = append(dest, err)
		}
	}
	return dest
}

// FromSlice combines a slice of errors into a single error. If the list is
// empty or all the errors in it are nil, a nil error is returned. If the list
// contains only a single error, the error is returned as-is.
//
// If any error in the list satisfies the following interface, errors from it
// will be flattened. This also applies to errors found nested inside other
// ErrorGroups.
//
// 	type ErrorGroup interface {
// 		Causes() []error
// 	}
//
// Returns nil if the error list is empty.
func FromSlice(errors []error, opts ...Option) error {
	errors = flatten(errors)
	switch len(errors) {
	case 0:
		return nil
	case 1:
		return errors[0]
	}

	err := multiError{Errors: errors}
	for _, opt := range opts {
		opt.apply(&err)
	}
	return &err
}

// Combine combines the given collection of errors together. nil values will
// be ignored.
//
// This may be used to combine errors together from operations that may fail
// independently.
//
// 	multierr.Combine(
// 		reader.Close(),
// 		writer.Close(),
// 	)
//
// This may also be used to record failure of deferred operations without
// losing information about the original error.
//
// 	func someFunc(...) (err error) {
// 		f := open(...)
// 		defer func() {
// 			err = multierr.Combine(err, f.Close())
// 		}()
// 		// ...
// 	}
//
// Combine does not allow customizing the error message. Use FromSlice if you
// need to customize the error message.
func Combine(errors ...error) error {
	return FromSlice(errors)
}

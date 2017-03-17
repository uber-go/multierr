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

import (
	"bytes"
	"io"
	"strings"
	"sync"
)

const (
	// Amount of space we reserve in a slice when flattening nested errorGroups.
	_errorBuffer = 8

	// Prefix for multiError messages
	_prefix = "the following errors occurred:"
)

// _bufferPool is a pool of bytes.Buffer objects
var _bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// errorGroup is an interface implemented by any error type which combines one
// or more errors.
type errorGroup interface {
	// Causes returns the list of errors that are wrapped by this ErrorGroup.
	Causes() []error
}

type multiError []error

func (me multiError) String() string {
	return me.Error()
}

func (me multiError) Causes() []error {
	return []error(me)
}

func (me multiError) Error() string {
	errLineIndent := strings.Repeat(" ", 4)

	buff := _bufferPool.Get().(*bytes.Buffer)
	buff.Reset()

	buff.WriteString(_prefix)
	for _, err := range me {
		buff.WriteString("\n -  ")
		writeWithPrefix(buff, errLineIndent, err.Error())
	}

	result := buff.String()
	_bufferPool.Put(buff)
	return result
}

// Writes s to the writer with the given prefix added before each line after
// the first.
func writeWithPrefix(w io.Writer, prefix, s string) error {
	first := true
	for len(s) > 0 {
		if first {
			first = false
		} else if _, err := io.WriteString(w, prefix); err != nil {
			return err
		}

		idx := strings.IndexByte(s, '\n')
		if idx < 0 {
			idx = len(s) - 1
		}

		if _, err := io.WriteString(w, s[:idx+1]); err != nil {
			return err
		}

		s = s[idx+1:]
	}
	return nil
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
// If an error in the list satisfies the following interface, it is squashed
// away and errors from it are flattened. This also applies to errors found
// nested inside other
// ErrorGroups.
//
// 	type ErrorGroup interface {
// 		Causes() []error
// 	}
func FromSlice(errors []error) error {
	errors = flatten(errors)
	switch len(errors) {
	case 0:
		return nil
	case 1:
		return errors[0]
	}
	return multiError(errors)
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

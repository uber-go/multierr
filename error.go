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
	"fmt"
	"io"
	"strings"
	"sync"
)

// Amount of space we reserve in a slice when flattening nested errorGroups.
const _errorBuffer = 8

var (
	// Separator for single-line error messages.
	_singlelineSeparator = []byte("; ")

	_newline = []byte("\n")

	// Prefix for multi-line messages
	_prefix = []byte("the following errors occurred:")

	// Prefix for the first and following lines of an item in a list of
	// multi-line error messages.
	//
	// For example, if a single item is:
	//
	// 	foo
	// 	bar
	//
	// It will become,
	//
	// 	 -  foo
	// 	    bar
	_listDash   = []byte(" -  ")
	_listIndent = []byte("    ")
)

// _bufferPool is a pool of bytes.Buffers.
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
	buff := _bufferPool.Get().(*bytes.Buffer)
	buff.Reset()

	me.writeSingleline(buff)

	result := buff.String()
	_bufferPool.Put(buff)
	return result
}

func (me multiError) Format(f fmt.State, c rune) {
	if c == 'v' && f.Flag('+') {
		me.writeMultiline(f)
	} else {
		me.writeSingleline(f)
	}
}

func (me multiError) writeSingleline(w io.Writer) error {
	first := true
	for _, item := range me {
		if first {
			first = false
		} else {
			if _, err := w.Write(_singlelineSeparator); err != nil {
				return err
			}
		}

		if _, err := io.WriteString(w, item.Error()); err != nil {
			return err
		}
	}
	return nil
}

func (me multiError) writeMultiline(w io.Writer) error {
	if _, err := w.Write(_prefix); err != nil {
		return err
	}

	for _, item := range me {
		if _, err := w.Write(_newline); err != nil {
			return err
		}

		if _, err := w.Write(_listDash); err != nil {
			return err
		}

		if err := writeWithPrefix(w, _listIndent, item.Error()); err != nil {
			return err
		}
	}

	return nil
}

// Writes s to the writer with the given prefix added before each line after
// the first.
func writeWithPrefix(w io.Writer, prefix []byte, s string) error {
	first := true
	for len(s) > 0 {
		if first {
			first = false
		} else if _, err := w.Write(prefix); err != nil {
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
//
// If multiple errors were found, the error returned by this function will
// result in a multi-line message if "%+v" is used with fmt.Printf and
// friends.
//
// 	fmt.Sprintf("%+v", multierr.FromSlice(errors))
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
// 		pipe.Close(),
// 	)
func Combine(errors ...error) error {
	return FromSlice(errors)
}

// Append appends the given error to the destination. Either error value may
// be nil or an ErrorGroup.
//
// This function is a specialization of Combine for the common case where
// there are only two errors.
//
// 	err = multierr.Append(reader.Close(), writer.Close())
//
// This may be used to record failure of deferred operations without losing
// information about the original error.
//
// 	func doSomething(..) (err error) {
// 		f := acquireResource()
// 		defer func() {
// 			err = multierr.Append(err, f.Close())
// 		}()
func Append(dest error, err error) error {
	switch {
	case dest == nil:
		return err
	case err == nil:
		return dest
	}

	errors := [2]error{dest, err}
	return FromSlice(errors[0:])
}

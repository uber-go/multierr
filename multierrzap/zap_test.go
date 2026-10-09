// Copyright (c) 2024 Uber Technologies, Inc.
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

package multierrzap_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/multierr"
	"go.uber.org/multierr/multierrzap"
	"go.uber.org/zap/zapcore"
)

type customObjectErr struct {
	msg string
}

func (c customObjectErr) Error() string { return c.msg }
func (c customObjectErr) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("msg", c.msg)
	return nil
}

type customArrayErr struct {
	items []string
}

func (c customArrayErr) Error() string { return "array error" }
func (c customArrayErr) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, item := range c.items {
		enc.AppendString(item)
	}
	return nil
}

type failingArrayErr struct {
	failMsg string
}

func (f failingArrayErr) Error() string { return f.failMsg }
func (f failingArrayErr) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	return errors.New(f.failMsg)
}

type failingObjectErr struct {
	failMsg string
}

func (f failingObjectErr) Error() string { return f.failMsg }
func (f failingObjectErr) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	return errors.New(f.failMsg)
}

func TestErrorsArrayMarshaler(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		ea := multierrzap.Errors(nil)
		assert.Empty(t, ea)

		enc := zapcore.NewMapObjectEncoder()
		err := enc.AddArray("errors", ea)
		require.NoError(t, err)
		assert.Equal(t, []interface{}{}, enc.Fields["errors"])
	})

	t.Run("single error", func(t *testing.T) {
		ea := multierrzap.Errors(errors.New("foo"))
		assert.Len(t, ea, 1)

		enc := zapcore.NewMapObjectEncoder()
		err := enc.AddArray("errors", ea)
		require.NoError(t, err)
		assert.Equal(t, []interface{}{"foo"}, enc.Fields["errors"])
	})

	t.Run("combined multierr", func(t *testing.T) {
		combined := multierr.Combine(
			errors.New("first"),
			errors.New("second"),
			nil,
			errors.New("third"),
		)
		ea := multierrzap.Errors(combined)
		require.Len(t, ea, 3)

		enc := zapcore.NewMapObjectEncoder()
		err := enc.AddArray("errors", ea)
		require.NoError(t, err)
		assert.Equal(t, []interface{}{"first", "second", "third"}, enc.Fields["errors"])
	})

	t.Run("Inspect equivalent to Errors", func(t *testing.T) {
		combined := multierr.Combine(
			errors.New("first"),
			errors.New("second"),
		)
		ea := multierrzap.Inspect(combined)
		require.Len(t, ea, 2)

		enc := zapcore.NewMapObjectEncoder()
		err := enc.AddArray("errors", ea)
		require.NoError(t, err)
		assert.Equal(t, []interface{}{"first", "second"}, enc.Fields["errors"])
	})

	t.Run("custom object and array error elements", func(t *testing.T) {
		combined := multierr.Combine(
			customObjectErr{msg: "obj1"},
			customArrayErr{items: []string{"a", "b"}},
		)
		ea := multierrzap.Errors(combined)
		require.Len(t, ea, 2)

		enc := zapcore.NewMapObjectEncoder()
		err := enc.AddArray("errors", ea)
		require.NoError(t, err)

		expected := []interface{}{
			map[string]interface{}{"msg": "obj1"},
			[]interface{}{"a", "b"},
		}
		assert.Equal(t, expected, enc.Fields["errors"])
	})

	t.Run("direct ErrorArray with nil element", func(t *testing.T) {
		ea := multierrzap.ErrorArray{
			errors.New("valid"),
			nil,
			errors.New("another"),
		}
		enc := zapcore.NewMapObjectEncoder()
		err := enc.AddArray("errors", ea)
		require.NoError(t, err)
		assert.Equal(t, []interface{}{"valid", "another"}, enc.Fields["errors"])
	})

	t.Run("error propagation from nested array marshaler", func(t *testing.T) {
		ea := multierrzap.ErrorArray{
			failingArrayErr{failMsg: "nested array failure"},
		}
		enc := zapcore.NewMapObjectEncoder()
		err := enc.AddArray("errors", ea)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nested array failure")
	})

	t.Run("error propagation from nested object marshaler", func(t *testing.T) {
		ea := multierrzap.ErrorArray{
			failingObjectErr{failMsg: "nested object failure"},
		}
		enc := zapcore.NewMapObjectEncoder()
		err := enc.AddArray("errors", ea)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nested object failure")
	})
}

func TestFieldConstructor(t *testing.T) {
	combined := multierr.Combine(
		errors.New("read failed"),
		errors.New("write failed"),
	)
	fld := multierrzap.Field("multierrs", combined)
	assert.Equal(t, "multierrs", fld.Key)
	assert.Equal(t, zapcore.ArrayMarshalerType, fld.Type)

	enc := zapcore.NewMapObjectEncoder()
	fld.AddTo(enc)
	assert.Equal(t, []interface{}{"read failed", "write failed"}, enc.Fields["multierrs"])
}

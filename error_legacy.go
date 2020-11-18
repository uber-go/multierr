// +build legacy

package multierr

import (
	"bytes"
	"sync"
)

// _bufferPool is a pool of bytes.Buffers.
var _bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func (merr *multiError) oldError() string {
	if merr == nil {
		return ""
	}

	buff := _bufferPool.Get().(*bytes.Buffer)
	buff.Reset()

	merr.writeSingleline(buff)

	result := buff.String()
	_bufferPool.Put(buff)
	return result
}

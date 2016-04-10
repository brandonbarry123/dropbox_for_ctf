// SUPPORT CODE
//
// You shouldn't need to alter
// the contents of this file

package pool

import (
	"bytes"
	"sync"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func GetBuffer() *bytes.Buffer {
	b := bufferPool.Get().(*bytes.Buffer)
	return b
}

func PutBuffer(b *bytes.Buffer) {
	b.Reset()
	bufferPool.Put(b)
}

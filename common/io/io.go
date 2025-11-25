package io

import (
	"io"
	"koria-core/common/bufpool"
)

// Copy оптимизированная версия io.Copy с buffer pooling
func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	buf := bufpool.LargePool.Get()
	defer bufpool.LargePool.Put(buf)

	return io.CopyBuffer(dst, src, buf)
}

// CopyN оптимизированная версия io.CopyN
func CopyN(dst io.Writer, src io.Reader, n int64) (written int64, err error) {
	buf := bufpool.LargePool.Get()
	defer bufpool.LargePool.Put(buf)

	written, err = io.CopyBuffer(dst, io.LimitReader(src, n), buf)
	return
}

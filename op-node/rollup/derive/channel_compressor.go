package derive

import (
	"bytes"
	"compress/zlib"
)

type ChannelCompressor struct {
	maxFrameSize uint64

	compress        *zlib.Writer
	buf             bytes.Buffer
	flushedCompress *zlib.Writer
	flushedBuf      bytes.Buffer
}

func NewChannelCompressor(maxFrameSize uint64) (*ChannelCompressor, error) {
	c := &ChannelCompressor{
		maxFrameSize: maxFrameSize,
	}

	var err error
	c.compress, err = zlib.NewWriterLevel(&c.buf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}

	c.flushedCompress, err = zlib.NewWriterLevel(&c.flushedBuf, zlib.BestCompression)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *ChannelCompressor) Reset() {
	c.buf.Reset()
	c.compress.Reset(&c.buf)
	c.flushedBuf.Reset()
	c.flushedCompress.Reset(&c.flushedBuf)
}

func (c *ChannelCompressor) Len() int {
	return c.buf.Len()
}

func (c *ChannelCompressor) Read(p []byte) (n int, err error) {
	return c.buf.Read(p)
}

func (c *ChannelCompressor) Write(b []byte) (int, error) {
	_, err := c.flushedCompress.Write(b)
	if err != nil {
		return 0, err
	}
	err = c.flushedCompress.Flush()
	if err != nil {
		return 0, err
	}
	if c.buf.Len() > 0 && uint64(c.flushedBuf.Len()) > c.maxFrameSize {
		// we've written some data already, and writing more would cause
		// the approx. compressed size to be greater than the max
		return 0, ErrMaxFrameSizeReached
	}
	return c.compress.Write(b)
}

func (c *ChannelCompressor) Flush() error {
	return c.compress.Flush()
}

func (c *ChannelCompressor) Close() error {
	return c.compress.Close()
}

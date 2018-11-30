package hls

import (
	"bufio"
	"os"

	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/format/ts"
)

// segment is a ts muxer writing to a file.
type segment struct {
	file *os.File
	buf  *bufio.Writer
	mux  *ts.Muxer
}

func (seg *segment) WriteHeader(streams []av.CodecData) error {
	return seg.mux.WriteHeader(streams)
}

func (seg *segment) WriteTrailer() error {
	return seg.mux.WriteTrailer()
}

func (seg *segment) WritePacket(pkt av.Packet) error {
	return seg.mux.WritePacket(pkt)
}

func (seg *segment) Close() error {
	if err := seg.buf.Flush(); err != nil {
		return err
	}
	if err := seg.file.Close(); err != nil {
		return err
	}
	return nil
}

func createSegment(fn string) (*segment, error) {
	file, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	buf := bufio.NewWriter(file)
	mux := ts.NewMuxer(buf)
	return &segment{file, buf, mux}, nil
}

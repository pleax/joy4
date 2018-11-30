package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/nareix/joy4/av/avutil"
	"github.com/nareix/joy4/format/hls"
	"github.com/nareix/joy4/format/rtmp"
)

func main() {
	// rtmp.Debug = true

	rtmpsv := rtmp.Server{
		Addr: "localhost:1935",
		HandlePublish: func(conn *rtmp.Conn) {
			fmt.Println(conn.URL)

			hls := hls.NewHLSMuxer("test", "/tmp/hls-chunks/1")

			if err := avutil.CopyFile(hls, conn); err != nil {
				panic(err)
			}

			if err := conn.Close(); err != nil {
				panic(err)
			}
		},
	}

	rtmpsv.ListenAndServe()

	// ffmpeg -re -i movie.flv -c copy -f flv rtmp://localhost/movie
	// ffmpeg -f avfoundation -i "0:0" .... -f flv rtmp://localhost/screen
	// ffplay http://localhost:8089/movie
	// ffplay http://localhost:8089/screen

}

type SegmentStorage interface {
	Playlist() io.WriteCloser
	Segment(index uint) io.WriteCloser
}

type nopWriteCloser struct {
	w io.Writer
}

func (nwc nopWriteCloser) Write(p []byte) (n int, err error) {
	return nwc.Write(p)
}

func (nwc nopWriteCloser) Close() error {
	return nil
}

type testSegmentStorage struct{}

func (t testSegmentStorage) Playlist() io.WriteCloser {
	return nopWriteCloser{os.Stdout} // we don't want to accidentally close stdout
}

func (t testSegmentStorage) Segment(index int) io.WriteCloser {
	return nopWriteCloser{ioutil.Discard}
}

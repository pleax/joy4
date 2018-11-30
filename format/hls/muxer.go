package hls

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/grafov/m3u8"
	"github.com/nareix/joy4/av"
)

type HLSMuxer struct {
	Name            string
	Path            string
	SegmentDuration time.Duration // stream will be cut on first keyframe after `SegmentDuration` passed

	playlist *m3u8.MediaPlaylist

	streams     []av.CodecData
	timeHorizon time.Duration

	segmentIndex int
	segmentStart time.Duration
	*segment
}

// NewHLSMuxer creates new hls muxer with name and path to directory where segments should be stored.
func NewHLSMuxer(name string, path string) *HLSMuxer {
	mux := &HLSMuxer{
		Name:            name,
		Path:            path,
		SegmentDuration: 5 * time.Second,
	}
	mux.SetMaxPlaylistItems(5)
	return mux
}

// SetMaxPlaylistItems sets new playlist max items number.
// Should only be used during initial configuration of the hls muxer.
func (hls *HLSMuxer) SetMaxPlaylistItems(n uint) {
	playlist, err := m3u8.NewMediaPlaylist(n, n+3)
	if err != nil {
		panic(err)
	}
	hls.playlist = playlist
}

func (hls HLSMuxer) segmentFilename() string {
	return fmt.Sprintf("%s%d.ts", hls.Name, hls.segmentIndex)
}

func (hls HLSMuxer) segmentPath() string {
	return filepath.Join(hls.Path, hls.segmentFilename())
}

func (hls HLSMuxer) playlistFilename() string {
	return fmt.Sprintf("%s.m3u8", hls.Name)
}

func (hls HLSMuxer) playlistPath() string {
	return filepath.Join(hls.Path, hls.playlistFilename())
}

func (hls *HLSMuxer) rollPlaylist(duration time.Duration) *m3u8.MediaSegment {
	seg := &m3u8.MediaSegment{
		URI:      hls.segmentFilename(),
		Duration: float64(duration) / float64(time.Second),
	}
	return hls.playlist.RollSegment(seg)
}

func (hls *HLSMuxer) writePlaylist() error {
	return writeAtomically(hls.playlistPath(), hls.playlist.Encode())
}

func (hls *HLSMuxer) startSegment() error {
	seg, err := createSegment(hls.segmentPath())
	if err != nil {
		return err
	}
	hls.segment = seg
	if err := seg.WriteHeader(hls.streams); err != nil {
		return err
	}
	return nil
}

func (hls *HLSMuxer) finishSegment() error {
	seg := hls.segment
	if err := seg.WriteTrailer(); err != nil {
		return err
	}
	if err := seg.Close(); err != nil {
		return err
	}
	hls.segment = nil
	return nil
}

func (hls *HLSMuxer) WriteHeader(streams []av.CodecData) error {
	hls.streams = streams
	return nil
}

func (hls *HLSMuxer) WriteTrailer() error {
	if err := hls.finishSegment(); err != nil {
		return err
	}

	hls.rollPlaylist(hls.timeHorizon - hls.segmentStart)
	hls.playlist.Close()
	if err := hls.writePlaylist(); err != nil {
		return err
	}
	hls.playlist = nil

	return nil
}

func (hls *HLSMuxer) WritePacket(pkt av.Packet) error {
	/*
		fmt.Println(hls.streams[pkt.Idx].Type())
		fmt.Println(pkt.IsKeyFrame)
		fmt.Println(pkt.Time)
		fmt.Println(pkt.CompositionTime)
	*/

	pktpts := pkt.Time + pkt.CompositionTime

	// is correct segment duration important?
	if pktpts > hls.timeHorizon {
		hls.timeHorizon = pktpts
	}

	if dur := pktpts - hls.segmentStart; hls.segment != nil && pkt.IsKeyFrame && dur > hls.SegmentDuration {
		log.Println("Closing segment:", hls.segmentPath())
		log.Println("Segment duration:", dur)

		if err := hls.finishSegment(); err != nil {
			return err
		}

		// TODO: make segmenter and playlist generator different goroutines

		outdated := hls.rollPlaylist(dur)
		if err := hls.writePlaylist(); err != nil {
			return err
		}
		if outdated != nil {
			log.Println("Cleaning up segment:", outdated.URI)
			outdatedPath := filepath.Join(hls.Path, outdated.URI)
			if err := os.Remove(outdatedPath); err != nil {
				return err
			}
		}

		hls.segmentIndex++
	}

	if hls.segment == nil && pkt.IsKeyFrame {
		hls.segmentStart = pktpts

		log.Println("Opening segment:", hls.segmentPath())

		if err := hls.startSegment(); err != nil {
			return err
		}
	}

	if seg := hls.segment; seg != nil {
		if err := seg.mux.WritePacket(pkt); err != nil {
			return err
		}
	}

	return nil
}

package hls

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func writeAtomically(filename string, src io.Reader) error {
	dir, fn := filepath.Split(filename)

	tmpf, err := ioutil.TempFile(dir, fn)
	if err != nil {
		return err
	}
	defer func() {
		if tmpf != nil {
			tmpf.Close()
			os.Remove(tmpf.Name())
		}
	}()

	if _, err := io.Copy(tmpf, src); err != nil {
		return err
	}
	if err := tmpf.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpf.Name(), filename); err != nil {
		return err
	}

	tmpf = nil // to prevent redundant cleanup

	return nil
}

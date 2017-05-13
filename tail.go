package tail

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

const (
	// FollowDisable is follow mode for disabled
	FollowDisable = iota
	// FollowDescriptor is follow mode for descriptor
	FollowDescriptor
	// FollowName is follow mode for Name
	FollowName
)

type fileInfo struct {
	Inode uint64
	Dev   uint64
	Size  int64
}

func newFileInfo(fi os.FileInfo) (*fileInfo, error) {
	sys := fi.Sys()
	if sys == nil {
		return nil, errors.New("failed to get sys from FileInfo")
	}
	st, ok := sys.(*syscall.Stat_t)
	if !ok {
		return nil, errors.New("sys is not syscall.Stat_t")
	}

	return &fileInfo{
		Inode: st.Ino,
		Dev:   uint64(st.Dev),
		Size:  st.Size,
	}, nil

}

// File represent File for tail
type File struct {
	*os.File
	Follow        int
	NamePattern   string
	SleepInterval time.Duration

	notChanged int
	lastPos    int64
}

// IsInaccessible returns bool that file is inaccessible
func (f *File) IsInaccessible() (bool, error) {
	statA, err := os.Stat(f.Name())
	if err != nil {
		return false, fmt.Errorf("failed to stat for name: %s", err)
	}
	statB, err := f.Stat()
	if err != nil {
		return false, fmt.Errorf("failed to stat for fd: %s", err)
	}
	fiA, err := newFileInfo(statA)
	if err != nil {
		return false, fmt.Errorf("failed to get sys for name: %s", err)
	}
	fiB, err := newFileInfo(statB)
	if err != nil {
		return false, fmt.Errorf("failed to get sys for fd: %s", err)
	}
	if fiA.Dev != fiB.Dev {
		// device is changed
		return true, nil
	}
	if fiA.Inode != fiB.Inode {
		// inode is changed
		return true, nil
	}
	if fiA.Size < f.lastPos {
		// truncated
		return true, nil
	}
	return false, nil
}
func (f *File) Read(p []byte) (int, error) {
	for {
		n, err := f.File.Read(p)
		f.lastPos += int64(n)
		if err != io.EOF {
			return n, err
		}
		if f.Follow == FollowDisable {
			return n, err
		}
		time.Sleep(f.SleepInterval)
		if f.Follow == FollowDescriptor {
			continue
		}
		f.notChanged++
		if f.notChanged > 4 {
			continue
		}
		f.notChanged = 0

		b, err := f.IsInaccessible()
		if err != nil {
			return 0, err
		}
		if b {
			f.reopen()
		}
	}
}

func (f *File) reopen() error {
	newf, err := os.OpenFile(f.Name(), os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	f.File.Close()
	f.File = newf
	f.lastPos = 0
	return nil
}

// Open returns tailed file
func Open(filename string) (*File, error) {
	f, err := os.OpenFile(filename, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	n, err := f.Seek(0, os.SEEK_END)
	if err != nil {
		return nil, err
	}
	tf := &File{
		File:          f,
		Follow:        FollowDisable,
		SleepInterval: time.Second,
		lastPos:       n,
	}
	return tf, nil
}

// OpenDescriptor opens file for tail by descriptor
func OpenDescriptor(filename string) (*File, error) {
	f, err := Open(filename)
	if err != nil {
		return nil, err
	}
	f.Follow = FollowDescriptor
	return f, nil
}

// OpenName opens file for tail by name
func OpenName(filename string) (*File, error) {
	f, err := Open(filename)
	if err != nil {
		return nil, err
	}
	f.Follow = FollowName
	f.NamePattern = filename
	return f, nil
}

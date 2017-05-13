package tail

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func checkRead(t *testing.T, err error, gotSize, expectedSize int, gotBytes, expectedBytes []byte) {
	if err != nil {
		t.Errorf("failed to read file: %s", err)
	}
	if gotSize != expectedSize {
		t.Errorf("invalid read size: got=%d, expected=%d", gotSize, expectedSize)
	}
	if bytes.Compare(gotBytes, expectedBytes) != 0 {
		t.Errorf("invalid data: got=`%s`, expected=`%s`", string(gotBytes), string(expectedBytes))
	}
}

func TestTailOpen(t *testing.T) {
	f, err := Open("nosuchfile")
	if f != nil {
		t.Errorf("file found")
	}
	if err == nil {
		t.Errorf("no error reported by nosuchfile")
	}
}

func TestDescriptor(t *testing.T) {
	f, err := ioutil.TempFile(".", "test_data")
	if err != nil {
		t.Errorf("failed to create tempfile: %s", err)
	}
	filename := f.Name()
	defer func() {
		os.Remove(filename)
	}()

	b := make([]byte, 8)
	tf, err := OpenDescriptor(filename)
	if err != nil {
		t.Errorf("failed to open file: %s", err)
	}
	tf.SleepInterval = time.Microsecond

	// first read
	f.WriteString("123456789")
	n, err := tf.Read(b)
	checkRead(t, err, n, 8, b, []byte("12345678"))

	// second read
	n, err = tf.Read(b)
	checkRead(t, err, n, 1, b[0:1], []byte("9"))

	// appended data
	go func() {
		time.Sleep(time.Millisecond)
		f.WriteString("10")
		f.Sync()
	}()
	n, err = tf.Read(b)
	checkRead(t, err, n, 2, b[0:2], []byte("10"))
}

func TestDescriptorUnlink(t *testing.T) {
	f, err := ioutil.TempFile(".", "test_data")
	if err != nil {
		t.Errorf("failed to create tempfile: %s", err)
	}
	filename := f.Name()

	tf, err := OpenDescriptor(filename)
	if err != nil {
		t.Errorf("failed to open file: %s", err)
	}
	tf.SleepInterval = time.Microsecond

	os.Remove(filename)

	b := make([]byte, 4)
	f.WriteString("test")
	n, err := tf.Read(b)
	checkRead(t, err, n, 4, b, []byte("test"))
}

func TestName(t *testing.T) {
	f, err := ioutil.TempFile(".", "test_data")
	if err != nil {
		t.Errorf("failed to create tempfile: %s", err)
	}
	filename := f.Name()

	tf, err := OpenName(filename)
	if err != nil {
		t.Errorf("failed to open file: %s", err)
	}
	tf.SleepInterval = time.Microsecond
	b := make([]byte, 8)

	f.WriteString("1")

	n, err := tf.Read(b)
	checkRead(t, err, n, 1, b[0:1], []byte("1"))
	f.Close()
	os.Remove(filename)

	f, _ = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	defer func() {
		os.Remove(filename)
	}()
	f.WriteString("2")
	f.Sync()
	n, err = tf.Read(b)
	checkRead(t, err, n, 1, b[0:1], []byte("2"))
}

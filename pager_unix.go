// unix implementation of the `pager` interface (the file reader) with POSIX
// advisory locking

package sqlittle

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

const (
	seek_set            = 0 // should be defined in syscall
	sqlite_pending_byte = 0x40000000
	sqlite_shared_first = sqlite_pending_byte + 2
	sqlite_shared_size  = 510
)

var (
	ErrFileTruncated = errors.New("file truncated")
)

type filePager struct {
	f        *os.File
	readLock *unix.Flock_t
}

func newFilePager(file string) (*filePager, error) {
	f, err := os.Open(file)
	return &filePager{f: f}, err
}

func (f *filePager) header() ([headerSize]byte, error) {
	buf := [headerSize]byte{}
	n, err := f.f.ReadAt(buf[:], 0)
	if n != headerSize {
		return buf, ErrFileTruncated
	}
	return buf, err
}

func (f *filePager) page(id int, pagesize int) ([]byte, error) {
	buf := make([]byte, pagesize)
	// pages start counting at 1
	n, err := f.f.ReadAt(buf[:], int64(id-1)*int64(pagesize))
	if err != nil {
		return buf, err
	}
	if n != len(buf) {
		return buf, ErrFileTruncated
	}
	return buf, nil
}

func (f *filePager) lock(flock *unix.Flock_t) error {
	return unix.FcntlFlock(f.f.Fd(), unix.F_SETLK, flock)
}

func (f *filePager) RLock() error {
	// Set a 'SHARED' lock, following unixLock() logic from sqlite3.c

	if f.readLock != nil {
		return errors.New("trying to lock a locked lock") // panic?
	}

	// - get PENDING lock
	pending := &unix.Flock_t{
		Type:   unix.F_RDLCK,
		Whence: seek_set,
		Start:  sqlite_pending_byte,
		Len:    1,
	}
	if err := f.lock(pending); err != nil {
		return err
	}

	defer func() {
		// - drop the pending lock. No idea what to do with the error :/
		pending.Type = unix.F_UNLCK
		f.lock(pending)
	}()

	// Get the read-lock
	read := &unix.Flock_t{
		Type:   unix.F_RDLCK,
		Whence: seek_set,
		Start:  sqlite_shared_first,
		Len:    sqlite_shared_size,
	}
	if err := f.lock(read); err != nil {
		return err
	}
	f.readLock = read
	return nil
}

func (f *filePager) RUnlock() error {
	if f.readLock == nil {
		return errors.New("trying to unlock an unlocked lock") // panic?
	}
	f.readLock.Type = unix.F_UNLCK
	f.lock(f.readLock)
	f.readLock = nil
	return nil
}

func (f *filePager) Close() error {
	return f.f.Close()
}
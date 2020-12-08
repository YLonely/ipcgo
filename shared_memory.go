package ipcgo

/*
#include<sys/shm.h>
*/
import "C"

import (
	"io"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

const (
	pageSize = 1 << 12
)

func NewSharedMemory(key int, size uint64, flag int) (*SharedMemory, error) {
	id, _, errno := syscall.Syscall(syscall.SYS_SHMGET, uintptr(key), uintptr(size), uintptr(flag))
	if errno != 0 {
		return nil, errors.Errorf("failed to create shm, error %s", errno.Error())
	}
	return &SharedMemory{
		shmid:      int(id),
		actualSize: roundUp(size, pageSize),
	}, nil
}

type SharedMemory struct {
	shmid      int
	actualSize uint64
	attached   bool
	addr       uintptr
	seek       uint64
}

func (s *SharedMemory) Size() uint64 {
	return s.actualSize
}

func (s *SharedMemory) Read(p []byte) (int, error) {
	if p == nil {
		return 0, errors.New("slice p is nil")
	}
	if !s.attached {
		return 0, errors.New("memory is not attached")
	}
	n := len(p)
	if n == 0 {
		return 0, nil
	}
	maxReadSize := min(s.actualSize-s.seek, uint64(n))
	if maxReadSize == 0 {
		return 0, io.EOF
	}
	var i uint64 = 0
	for i = 0; i < maxReadSize; i++ {
		p[i] = *(*byte)(unsafe.Pointer(s.addr + uintptr(s.seek)))
		s.seek++
	}
	return int(maxReadSize), nil
}

func (s *SharedMemory) Write(p []byte) (int, error) {
	if p == nil {
		return 0, errors.New("slice p is nil")
	}
	if !s.attached {
		return 0, errors.New("memory is not attached")
	}
	n := len(p)
	if n == 0 {
		return 0, nil
	}
	maxWriteSize := min(s.actualSize-s.seek, uint64(n))
	if maxWriteSize == 0 {
		return 0, errors.New("end of memory")
	}
	var i uint64 = 0
	for i = 0; i < maxWriteSize; i++ {
		*(*byte)(unsafe.Pointer(s.addr + uintptr(s.seek))) = p[i]
		s.seek++
	}
	return int(maxWriteSize), nil
}

func (s *SharedMemory) Seek(offset uint64, whence int) (int64, error) {
	switch whence {
	case io.SeekCurrent:
		if offset > s.actualSize {
			return int64(s.seek), errors.New("seek out of memory range")
		}
		s.seek = offset
	case io.SeekEnd:
		s.seek = s.actualSize
	case io.SeekStart:
		s.seek = 0
	default:
	}
	return int64(s.seek), nil
}

func (s *SharedMemory) Close() error {
	return s.detach()
}

func (s *SharedMemory) Attach(addr uintptr, flag int) error {
	if s.attached {
		return nil
	}
	shmAddr, _, errno := syscall.Syscall(syscall.SYS_SHMAT, uintptr(s.shmid), addr, uintptr(flag))
	if errno != 0 {
		return errors.Errorf("failed to attach memory, error %s", errno.Error())
	}
	s.addr = shmAddr
	s.attached = true
	return nil
}

func (s *SharedMemory) Stat() C.struct_shmid_ds {
	var stat C.struct_shmid_ds
	_, _, errno := syscall.Syscall(syscall.SYS_SHMCTL, uintptr(C.IPC_STAT), unsafe.Pointer(&stat), 0)
}

func (s *SharedMemory) detach() error {
	if !s.attached {
		return nil
	}
	_, _, errno := syscall.Syscall(syscall.SYS_SHMDT, uintptr(s.shmid), 0, 0)
	if errno != 0 {
		return errors.Errorf("failed to detach memory, error %s", errno.Error())
	}
	s.attached = false
	return nil
}

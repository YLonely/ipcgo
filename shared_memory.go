package ipcgo

/*
#include<sys/shm.h>
#include<sys/ipc.h>
*/
import "C"

import (
	"io"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

const (
	pageSize   = 1 << 12
	SHM_R      = C.SHM_R
	SHM_W      = C.SHM_W
	SHM_RDONLY = C.SHM_RDONLY
	SHM_RND    = C.SHM_RND
	SHM_REMAP  = C.SHM_REMAP
)

// C_shmid_ds is the struct from C
type C_shmid_ds struct {
	Shm_perm   C_ipc_perm
	Shm_segsz  uint64
	Shm_atime  int64
	Shm_dtime  int64
	Shm_ctime  int64
	Shm_cpid   int32
	Shm_lpid   int32
	Shm_nattch uint64
}

// C_ipc_perm is the struct from C
type C_ipc_perm struct {
	Key  int32
	Uid  uint32
	Gid  uint32
	Cuid uint32
	Cgid uint32
	Mode uint16
	Seq  uint16
}

// NewSharedMemory returns a new shared memory
func NewSharedMemory(key int, size uint64, flag int) (*SharedMemory, error) {
	if key == C.IPC_PRIVATE {
		return nil, errors.New("creating of private memory region is not supported")
	}
	flag |= C.IPC_CREAT | C.IPC_EXCL
	id, _, errno := syscall.Syscall(syscall.SYS_SHMGET, uintptr(key), uintptr(size), uintptr(flag))
	if errno != 0 {
		return nil, errors.Errorf("failed to create shm, error %s", errno.Error())
	}
	return &SharedMemory{
		shmid:      int(id),
		actualSize: roundUp(size, pageSize),
	}, nil
}

// GetSharedMemory returns a memory region already exists
func GetSharedMemory(key int) (*SharedMemory, error) {
	if key == C.IPC_PRIVATE {
		return nil, errors.New("getting a private memory region is not supported")
	}
	id, _, errno := syscall.Syscall(syscall.SYS_SHMGET, uintptr(key), 0, 0)
	if errno != 0 {
		return nil, errors.Errorf("failed to get shm, error %s", errno.Error())
	}
	ds, err := stat(int(id))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get info of shm")
	}
	return &SharedMemory{
		shmid:      int(id),
		actualSize: roundUp(ds.Shm_segsz, pageSize),
	}, nil
}

type SharedMemory struct {
	shmid      int
	actualSize uint64
	addr       uintptr
	seek       uint64
}

// Size returns the size of the shared memory, it is round up towards page size
func (s *SharedMemory) Size() uint64 {
	return s.actualSize
}

// ID returns the id of shared memory
func (s *SharedMemory) ID() int {
	return s.shmid
}

// Read reads content of memory to p
func (s *SharedMemory) Read(p []byte) (int, error) {
	if p == nil {
		return 0, errors.New("slice p is nil")
	}
	if s.addr == 0 {
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

// Write writes p to memory
func (s *SharedMemory) Write(p []byte) (int, error) {
	if p == nil {
		return 0, errors.New("slice p is nil")
	}
	if s.addr == 0 {
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

// Seek seeks inside the range of memory
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

// Close detaches the memory
func (s *SharedMemory) Close() error {
	return s.detach()
}

// Delete deletes the shared memory from system
func (s *SharedMemory) Delete() error {
	_, _, errno := syscall.Syscall(syscall.SYS_SHMCTL, uintptr(s.shmid), uintptr(C.IPC_RMID), 0)
	if errno != 0 {
		return errors.Errorf("failed to delete shm, error %s", errno.Error())
	}
	return nil
}

// Attach attaches the memory to process's virtual memory space
func (s *SharedMemory) Attach(addr uintptr, flag int) error {
	shmAddr, _, errno := syscall.Syscall(syscall.SYS_SHMAT, uintptr(s.shmid), addr, uintptr(flag))
	if errno != 0 {
		return errors.Errorf("failed to attach memory, error %s", errno.Error())
	}
	s.addr = shmAddr
	s.seek = 0
	return nil
}

// Stat returns the info of the shared memory
func (s *SharedMemory) Stat() (*C_shmid_ds, error) {
	return stat(s.shmid)
}

// SetStat sets the settable fields of shm
func (s *SharedMemory) SetStat(uid, gid *uint32, mode *uint16) error {
	var cstat C.struct_shmid_ds
	_, _, errno := syscall.Syscall(syscall.SYS_SHMCTL, uintptr(s.shmid), uintptr(C.IPC_STAT), uintptr(unsafe.Pointer(&cstat)))
	if errno != 0 {
		return errors.Errorf("failed to get stat, error %s", errno.Error())
	}
	if uid != nil {
		cstat.shm_perm.uid = C.uint(*uid)
	}
	if gid != nil {
		cstat.shm_perm.gid = C.uint(*gid)
	}
	if mode != nil {
		cstat.shm_perm.mode = C.ushort(*mode)
	}
	_, _, errno = syscall.Syscall(syscall.SYS_SHMCTL, uintptr(s.shmid), uintptr(C.IPC_SET), uintptr(unsafe.Pointer(&cstat)))
	if errno != 0 {
		return errors.Errorf("failed to set stat, error %s", errno.Error())
	}
	return nil
}

func stat(id int) (*C_shmid_ds, error) {
	var cstat C.struct_shmid_ds
	_, _, errno := syscall.Syscall(syscall.SYS_SHMCTL, uintptr(id), uintptr(C.IPC_STAT), uintptr(unsafe.Pointer(&cstat)))
	if errno != 0 {
		return nil, errors.Errorf("failed to get stat, error %s", errno.Error())
	}
	return &C_shmid_ds{
		Shm_perm: C_ipc_perm{
			Key:  int32(cstat.shm_perm.__key),
			Uid:  uint32(cstat.shm_perm.uid),
			Gid:  uint32(cstat.shm_perm.gid),
			Cuid: uint32(cstat.shm_perm.cuid),
			Cgid: uint32(cstat.shm_perm.cgid),
			Mode: uint16(cstat.shm_perm.mode),
			Seq:  uint16(cstat.shm_perm.__seq),
		},
		Shm_segsz:  uint64(cstat.shm_segsz),
		Shm_atime:  int64(cstat.shm_atime),
		Shm_dtime:  int64(cstat.shm_dtime),
		Shm_ctime:  int64(cstat.shm_ctime),
		Shm_cpid:   int32(cstat.shm_cpid),
		Shm_lpid:   int32(cstat.shm_lpid),
		Shm_nattch: uint64(cstat.shm_nattch),
	}, nil

}

func (s *SharedMemory) detach() error {
	_, _, errno := syscall.Syscall(syscall.SYS_SHMDT, uintptr(s.addr), 0, 0)
	if errno != 0 {
		return errors.Errorf("failed to detach memory, error %s", errno.Error())
	}
	return nil
}

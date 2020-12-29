package ipcgo

/*
#include<sys/types.h>
#include<sys/ipc.h>
#include<sys/sem.h>

union semun{
	int val;
	struct semid_ds *buf;
	unsigned short *array;
	struct seminfo *__buf;
};
*/
import "C"
import (
	"reflect"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

type C_semid_ds struct {
	C_sem_perm C_ipc_perm
	Sem_otime  int64
	Sem_ctime  int64
	Sem_nsems  uint64
}

type C_sembuf struct {
	Sem_num uint16
	Sem_op  int16
	Sem_flg int16
}

const (
	GETVAL   = 12
	GETPID   = 11
	GETNCNT  = 14
	GETZCNT  = 15
	getAll   = 13
	setAll   = 17
	setVal   = 16
	SEM_UNDO = C.SEM_UNDO
)

// NewSemaphoreSet creates a new semaphore set
func NewSemaphoreSet(key, nsems, flag int) (*SemaphoreSet, error) {
	if key == C.IPC_PRIVATE {
		return nil, errors.New("creating a private semaphore set is not supported")
	}
	flag |= C.IPC_CREAT | C.IPC_EXCL
	id, _, errno := syscall.Syscall(syscall.SYS_SEMGET, uintptr(C.key_t(key)), uintptr(C.int(nsems)), uintptr(C.int(flag)))
	if errno != 0 {
		return nil, errors.Wrap(errno, "failed to create a new semaphore set")
	}
	return &SemaphoreSet{
		id:   int(id),
		size: nsems,
	}, nil
}

// GetSemaphoreSet returns a semaphore set already exists
func GetSemaphoreSet(key int) (*SemaphoreSet, error) {
	if key == C.IPC_PRIVATE {
		return nil, errors.New("getting a private semaphore set is not supported")
	}
	id, _, errno := syscall.Syscall(syscall.SYS_SEMGET, uintptr(C.key_t(key)), 0, 0)
	if errno != 0 {
		return nil, errors.Wrap(errno, "failed to get the semaphore set")
	}
	cstat, err := semStat(int(id))
	if err != nil {
		return nil, err
	}
	return &SemaphoreSet{
		id:   int(id),
		size: int(cstat.sem_nsems),
	}, nil
}

// SemaphoreSet represents a System V semaphore set
type SemaphoreSet struct {
	id   int
	size int
}

// ID returns the id of the semaphore set
func (sem *SemaphoreSet) ID() int {
	return sem.id
}

// Size returns the size of the semaphore set
func (sem *SemaphoreSet) Size() int {
	return sem.size
}

// Delete deletes the semaphore set from the system
func (sem *SemaphoreSet) Delete() error {
	_, _, errno := syscall.Syscall(syscall.SYS_SEMCTL, uintptr(sem.id), 0, uintptr(C.IPC_RMID))
	if errno != 0 {
		return errors.Wrap(errno, "failed to delete the semaphore set")
	}
	return nil
}

// Stat returns the info of the semaphore set
func (sem *SemaphoreSet) Stat() (*C_semid_ds, error) {
	cstat, err := semStat(sem.id)
	if err != nil {
		return nil, err
	}
	return &C_semid_ds{
		C_sem_perm: C_ipc_perm{
			Key:  int32(cstat.sem_perm.__key),
			Cgid: uint32(cstat.sem_perm.cgid),
			Cuid: uint32(cstat.sem_perm.cuid),
			Gid:  uint32(cstat.sem_perm.gid),
			Mode: uint16(cstat.sem_perm.mode),
			Seq:  uint16(cstat.sem_perm.__seq),
			Uid:  uint32(cstat.sem_perm.uid),
		},
		Sem_ctime: int64(cstat.sem_ctime),
		Sem_nsems: uint64(cstat.sem_nsems),
		Sem_otime: int64(cstat.sem_otime),
	}, nil
}

// SetStat changes some fields of the struct semid_ds of a semaphore set
func (sem *SemaphoreSet) SetStat(uid, gid *uint32, mode *uint16) error {
	cstat, err := semStat(sem.id)
	if err != nil {
		return err
	}
	if uid != nil {
		cstat.sem_perm.uid = C.uint(*uid)
	}
	if gid != nil {
		cstat.sem_perm.gid = C.uint(*gid)
	}
	if mode != nil {
		cstat.sem_perm.mode = C.ushort(*mode)
	}
	_, _, errno := syscall.Syscall6(syscall.SYS_SEMCTL, uintptr(C.int(sem.id)), 0, uintptr(C.IPC_SET), uintptr(unsafe.Pointer(cstat)), 0, 0)
	if errno != 0 {
		return errors.Wrap(errno, "failed to set stat")
	}
	return nil
}

// GetValue returns the value of ith semaphore, vType stands for the type of the value
func (sem *SemaphoreSet) GetValue(i, vType int) (int, error) {
	if i < 0 || i >= sem.size {
		return 0, errors.New("invalid index")
	}
	switch vType {
	case GETNCNT:
	case GETPID:
	case GETZCNT:
	case GETVAL:
	default:
		return 0, errors.New("invalid value type")
	}
	v, _, errno := syscall.Syscall(syscall.SYS_SEMCTL, uintptr(C.int(sem.id)), uintptr(C.int(i)), uintptr(C.int(vType)))
	if errno != 0 {
		return 0, errors.Wrap(errno, "failed to get value")
	}
	return int(v), nil
}

// SetValue sets the semaphore value
func (sem *SemaphoreSet) SetValue(i, v int) error {
	if i < 0 {
		return errors.New("invalid index")
	}
	_, _, errno := syscall.Syscall6(syscall.SYS_SEMCTL, uintptr(sem.id), uintptr(C.int(i)), uintptr(C.int(setVal)), uintptr(v), 0, 0)
	if errno != 0 {
		return errors.Wrap(errno, "failed to set value")
	}
	return nil
}

// SetAll sets the semval values for all semaphores of the set using values
func (sem *SemaphoreSet) SetAll(values []uint16) error {
	if len(values) != sem.size {
		return errors.Errorf("wrong values length, expect %d", sem.size)
	}
	header := (*reflect.SliceHeader)(unsafe.Pointer(&values))
	_, _, errno := syscall.Syscall6(syscall.SYS_SEMCTL, uintptr(sem.id), 0, uintptr(C.int(setAll)), header.Data, 0, 0)
	if errno != 0 {
		return errors.Wrap(errno, "failed to set all values")
	}
	return nil
}

// GetAll returns values of all semaphores
func (sem *SemaphoreSet) GetAll() ([]int, error) {
	var array C.ushort
	var innerValues []C.ushort
	var values []int
	_, _, errno := syscall.Syscall6(syscall.SYS_SEMCTL, uintptr(C.int(sem.id)), 0, uintptr(C.int(getAll)), uintptr(unsafe.Pointer(&array)), 0, 0)
	if errno != 0 {
		return nil, errors.Wrap(errno, "failed to get all values")
	}
	header := (*reflect.SliceHeader)(unsafe.Pointer(&innerValues))
	header.Data = uintptr(unsafe.Pointer(&array))
	header.Len = sem.size
	header.Cap = sem.size
	for _, v := range innerValues {
		values = append(values, int(v))
	}
	return values, nil
}

// Op performs operations on selected semaphores in the set
func (sem *SemaphoreSet) Op(ops []C_sembuf) error {
	sembufs := make([]C.struct_sembuf, 0, len(ops))
	for _, op := range ops {
		sembufs = append(sembufs, C.struct_sembuf{
			sem_num: C.ushort(op.Sem_num),
			sem_op:  C.short(op.Sem_op),
			sem_flg: C.short(op.Sem_flg),
		})
	}
	header := (*reflect.SliceHeader)(unsafe.Pointer(&sembufs))
	_, _, errno := syscall.Syscall(syscall.SYS_SEMOP, uintptr(sem.id), header.Data, uintptr(C.size_t(len(ops))))
	if errno != 0 {
		return errors.Wrap(errno, "failed to do ops")
	}
	return nil
}

func semStat(id int) (*C.struct_semid_ds, error) {
	var cstat C.struct_semid_ds
	_, _, errno := syscall.Syscall6(syscall.SYS_SEMCTL, uintptr(C.int(id)), 0, uintptr(C.IPC_STAT), uintptr(unsafe.Pointer(&cstat)), 0, 0)
	if errno != 0 {
		return nil, errors.Wrap(errno, "failed to get stat")
	}
	return &cstat, nil
}

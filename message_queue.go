package ipcgo

/*
#include<sys/msg.h>
#include<sys/ipc.h>
#include<sys/types.h>

#define MAX_LEN 8192

struct msgbuf{
	long mtype;
	char mtext[MAX_LEN];
};
*/
import "C"
import (
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
)

type C_msqid_ds struct {
	C_msg_perm   C_ipc_perm
	Msg_stime    int64
	Msg_rtime    int64
	Msg_ctime    int64
	__Msg_cbytes uint64
	Msg_qnum     uint64
	Msg_qbytes   uint64
	Msg_lspid    int32
	Msg_lrpid    int32
}

type Message struct {
	MType int64
	MText []byte
}

// NewMessageQueue creates a message queue
func NewMessageQueue(key int, flag int) (*MessageQueue, error) {
	if key == C.IPC_PRIVATE {
		return nil, errors.New("creating of private message queue is not supported")
	}
	flag |= C.IPC_CREAT | C.IPC_EXCL
	id, _, errno := syscall.Syscall(syscall.SYS_MSGGET, uintptr(C.key_t(key)), uintptr(C.int(flag)), 0)
	if errno != 0 {
		return nil, errors.Wrap(errno, "failed to create message queue")
	}
	return &MessageQueue{
		queueID: int(id),
	}, nil
}

// GetMessageQueue returns a message queue already exists
func GetMessageQueue(key int) (*MessageQueue, error) {
	if key == C.IPC_PRIVATE {
		return nil, errors.New("creating of private message queue is not supported")
	}
	id, _, errno := syscall.Syscall(syscall.SYS_MSGGET, uintptr(C.key_t(key)), 0, 0)
	if errno != 0 {
		return nil, errors.Wrap(errno, "failed to get message queue")
	}
	return &MessageQueue{
		queueID: int(id),
	}, nil
}

// MessageQueue represents a System V message queue
type MessageQueue struct {
	queueID int
}

const (
	maxLen      = C.MAX_LEN
	IPC_NOWAIT  = C.IPC_NOWAIT
	MSG_NOERROR = 010000
	MSG_EXCEPT  = 020000
	MSG_COPY    = 040000
)

// ID returns the id of the message queue
func (mq *MessageQueue) ID() int {
	return mq.queueID
}

// Stat returns the info of the message queue
func (mq *MessageQueue) Stat() (*C_msqid_ds, error) {
	var cstat C.struct_msqid_ds
	_, _, errno := syscall.Syscall(syscall.SYS_MSGCTL, uintptr(C.int(mq.queueID)), uintptr(C.IPC_STAT), uintptr(unsafe.Pointer(&cstat)))
	if errno != 0 {
		return nil, errors.Wrap(errno, "failed to get stat")
	}
	return &C_msqid_ds{
		C_msg_perm: C_ipc_perm{
			Key:  int32(cstat.msg_perm.__key),
			Uid:  uint32(cstat.msg_perm.uid),
			Gid:  uint32(cstat.msg_perm.gid),
			Cuid: uint32(cstat.msg_perm.cuid),
			Mode: uint16(cstat.msg_perm.mode),
			Seq:  uint16(cstat.msg_perm.__seq),
		},
		Msg_stime:    int64(cstat.msg_stime),
		Msg_rtime:    int64(cstat.msg_rtime),
		Msg_ctime:    int64(cstat.msg_ctime),
		__Msg_cbytes: uint64(cstat.__msg_cbytes),
		Msg_qnum:     uint64(cstat.msg_qnum),
		Msg_qbytes:   uint64(cstat.msg_qbytes),
		Msg_lspid:    int32(cstat.msg_lspid),
		Msg_lrpid:    int32(cstat.msg_lrpid),
	}, nil
}

// SetStat changes some fields of the struct msqid_ds of a message queue
func (mq *MessageQueue) SetStat(uid, gid *uint32, mode *uint16, msg_qbytes *uint64) error {
	var cstat C.struct_msqid_ds
	_, _, errno := syscall.Syscall(syscall.SYS_MSGCTL, uintptr(C.int(mq.queueID)), uintptr(C.IPC_STAT), uintptr(unsafe.Pointer(&cstat)))
	if errno != 0 {
		return errors.Wrap(errno, "failed to get stat")
	}
	if uid != nil {
		cstat.msg_perm.uid = C.uint(*uid)
	}
	if gid != nil {
		cstat.msg_perm.gid = C.uint(*gid)
	}
	if mode != nil {
		cstat.msg_perm.mode = C.ushort(*mode)
	}
	if msg_qbytes != nil {
		cstat.msg_qbytes = C.ulong(*msg_qbytes)
	}
	_, _, errno = syscall.Syscall(syscall.SYS_MSGCTL, uintptr(C.int(mq.queueID)), uintptr(C.IPC_SET), uintptr(unsafe.Pointer(&cstat)))
	if errno != 0 {
		return errors.Wrap(errno, "failed to set stat")
	}
	return nil
}

// Delete deletes message queue from system
func (mq *MessageQueue) Delete() error {
	_, _, errno := syscall.Syscall(syscall.SYS_MSGCTL, uintptr(C.int(mq.queueID)), uintptr(C.IPC_RMID), 0)
	if errno != 0 {
		return errors.Wrap(errno, "failed to delete message queue")
	}
	return nil
}

// Send sends a message to a message queue
func (mq *MessageQueue) Send(msg *Message, flag int) error {
	var cMsg C.struct_msgbuf
	if msg == nil {
		return errors.New("msg is a nil pointer")
	}
	if msg.MType < 0 {
		return errors.New("negative mtype is invalid")
	}
	n := len(msg.MText)
	if n > maxLen {
		return errors.Errorf("length of mtext %d exceeds the max value of length", n)
	}
	cMsg.mtype = C.long(msg.MType)
	for i, b := range msg.MText {
		cMsg.mtext[i] = C.char(b)
	}
	_, _, errno := syscall.Syscall6(syscall.SYS_MSGSND, uintptr(C.int(mq.queueID)), uintptr(unsafe.Pointer(&cMsg)), uintptr(C.size_t(n)), uintptr(C.int(flag)), 0, 0)
	if errno != 0 {
		return errors.Wrap(errno, "failed to send message")
	}
	return nil
}

// Receive receives a message from a message queue
func (mq *MessageQueue) Receive(msgSize uint64, msgType int64, flag int) (*Message, error) {
	var cMsg C.struct_msgbuf
	realSize, _, errno := syscall.Syscall6(syscall.SYS_MSGRCV, uintptr(C.int(mq.queueID)), uintptr(unsafe.Pointer(&cMsg)), uintptr(C.size_t(msgSize)), uintptr(C.long(msgType)), uintptr(C.int(flag)), 0)
	if errno != 0 {
		return nil, errors.Wrap(errno, "failed to receive message from message queue")
	}
	msg := &Message{
		MType: int64(cMsg.mtype),
	}
	text := make([]byte, int(realSize))
	for i := 0; i < int(realSize); i++ {
		text[i] = byte(cMsg.mtext[i])
	}
	msg.MText = text
	return msg, nil
}

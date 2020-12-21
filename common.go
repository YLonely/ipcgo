package ipcgo

const (
	pageSize = 1 << 12
)

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

func roundUp(num, multiple uint64) uint64 {
	return ((num + multiple - 1) / multiple) * multiple
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

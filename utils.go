package ipcgo

func roundUp(num, multiple uint64) uint64 {
	return ((num + multiple - 1) / multiple) * multiple
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

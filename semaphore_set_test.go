package ipcgo

import (
	"fmt"
	"testing"
)

func TestSemaphoreSet(t *testing.T) {
	key := 1234
	size := 10
	sem, err := NewSemaphoreSet(key, size, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer sem.Delete()
	var uid, gid uint32 = 111, 111
	err = sem.SetStat(&uid, &gid, nil)
	if err != nil {
		t.Fatal(err)
	}
	stat, err := sem.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if stat.C_sem_perm.Uid != uid || stat.C_sem_perm.Gid != gid {
		t.Fatalf("uid or git mismatch, uid %d, gid %d", stat.C_sem_perm.Uid, stat.C_sem_perm.Gid)
	}
	t.Log(stat)
	vals, err := sem.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	for i := range vals {
		if vals[i] != 0 {
			t.Fatalf("non-zero value at index %d", i)
		}
	}
	err = sem.SetValue(3, 123)
	if err != nil {
		t.Fatal(err)
	}
	ssem, err := GetSemaphoreSet(key)
	if err != nil {
		t.Fatal(err)
	}
	if ssem.Size() != size {
		t.Fatalf("wrong size, get %d", ssem.Size())
	}
	v, err := ssem.GetValue(3, GETVAL)
	if err != nil {
		t.Fatal(err)
	}
	if v != 123 {
		t.Fatalf("wrong value, get %d", v)
	}
	err = ssem.Op([]C_sembuf{
		{
			Sem_num: 3,
			Sem_op:  1,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	v, err = ssem.GetValue(3, GETVAL)
	if err != nil {
		t.Fatal(err)
	}
	if v != 124 {
		t.Fatalf("wrong increased value, get %d", v)
	}
	values := make([]uint16, size)
	for i := range values {
		values[i] = uint16(i)
	}
	err = ssem.SetAll(values)
	if err != nil {
		t.Fatal(err)
	}
	vs, err := ssem.GetAll()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(vs)
}

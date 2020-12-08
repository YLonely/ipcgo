package ipcgo

import (
	"fmt"
	"io"
	"testing"
)

func TestSharedMemory(t *testing.T) {
	key := 1234
	shm, err := NewSharedMemory(key, 1024, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = shm.Attach(0, 0); err != nil {
		t.Fatal(err)
	}
	str1 := "hello_world"
	str2 := "range_test"
	if _, err = fmt.Fprintln(shm, str1); err != nil {
		t.Fatal(err)
	}
	if _, err = shm.Seek(2048, io.SeekCurrent); err != nil {
		t.Fatal(err)
	}
	if _, err = fmt.Fprintln(shm, str2); err != nil {
		t.Fatal(err)
	}
	// read from shm
	if err = shm.Close(); err != nil {
		t.Fatal(err)
	}
	mm, err := GetSharedMemory(key)
	if err != nil {
		t.Fatal(err)
	}
	if err = mm.Attach(0, 0); err != nil {
		t.Fatal(err)
	}
	if info, err := mm.Stat(); err != nil {
		t.Fatal(err)
	} else {
		fmt.Println(info)
	}
	var str11, str22 string
	if _, err = fmt.Fscanln(mm, &str11); err != nil {
		t.Fatal(err)
	}
	if str11 != str1 {
		t.Fatal("scan from begining return " + str11)
	}
	mm.Seek(2048, io.SeekCurrent)
	if _, err = fmt.Fscanln(mm, &str22); err != nil {
		t.Fatal(err)
	}
	if str22 != str2 {
		t.Fatal("scan from 2048 return " + str22)
	}
	if err = mm.Close(); err != nil {
		t.Fatal(err)
	}
	if err = mm.Delete(); err != nil {
		t.Fatal(err)
	}
}

package ipcgo

import (
	"fmt"
	"syscall"
	"testing"

	"github.com/pkg/errors"
)

func TestMessageQueue(t *testing.T) {
	key := 1234
	sender, err := NewMessageQueue(key, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer sender.Delete()
	stat, err := sender.Stat()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(stat)
	if _, err = sender.Receive(100, 100, IPC_NOWAIT); err == nil {
		t.Fatal("receive a message from an empty queue")
	} else if !errors.Is(err, syscall.ENOMSG) {
		t.Fatalf("wrong error type: %s", err.Error())
	}
	text := "hello world"
	msg := Message{
		MType: 1024,
		MText: []byte(text),
	}
	if err = sender.Send(&msg, IPC_NOWAIT); err != nil {
		t.Fatal(err)
	}
	receiver, err := GetMessageQueue(key)
	if err != nil {
		t.Fatal(err)
	}
	m, err := receiver.Receive(uint64(len(text)+1), 1024, IPC_NOWAIT)
	if err != nil {
		t.Fatal(err)
	}
	if string(m.MText) != text {
		t.Fatalf("get %s, expect %s", m.MText, text)
	}
}

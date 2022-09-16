package etcd

import (
	"go.uber.org/zap"
	"testing"
	"time"
)

var endpoints = []string{"localhost:2379"}

func TestNewServiceRegister(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ser, err := NewServiceRegister(endpoints, "/web/node1", "localhost:8000", 5, logger)
	if err != nil {
		t.Fatal(err)
	}

	ser1, err := NewServiceRegister(endpoints, "/web/node2", "127.0.0.1:8000", 5, logger)
	if err != nil {
		t.Fatal(err)
	}

	go ser.ListenLeaseRespChan()
	go ser1.ListenLeaseRespChan()

	select {
	case <-time.After(180 * time.Second):
		ser.Close()
	}
}

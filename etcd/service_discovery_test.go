package etcd

import (
	"go.uber.org/zap"
	"log"
	"testing"
	"time"
)

func TestNewServiceDiscovery(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	ser := NewServiceDiscovery(endpoints, logger)
	defer ser.Close()
	ser.WatchService("/web/")
	ser.WatchService("/gRPC/")
	for {
		select {
		case <-time.Tick(5 * time.Second):
			log.Println(ser.GetServices())
		}
	}
}

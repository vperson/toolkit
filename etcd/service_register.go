package etcd

import (
	"context"
	"go.uber.org/zap"
	"time"

	"go.etcd.io/etcd/client/v3"
)

//ServiceRegister 创建租约注册服务
type ServiceRegister struct {
	cli     *clientv3.Client //etcd client
	leaseID clientv3.LeaseID //租约ID
	//租约keepalieve相应chan
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
	key           string //key
	val           string //value
	logger        *zap.Logger
}

//NewServiceRegister 新建注册服务
func NewServiceRegister(endpoints []string, key, val string, lease int64, logger *zap.Logger) (*ServiceRegister, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		logger.Fatal("new service register is error", zap.Error(err))
	}

	ser := &ServiceRegister{
		cli:    cli,
		key:    key,
		val:    val,
		logger: logger,
	}

	//申请租约设置时间keepalive
	if err := ser.putKeyWithLease(lease); err != nil {
		return nil, err
	}

	return ser, nil
}

//设置租约
func (s *ServiceRegister) putKeyWithLease(lease int64) error {
	//设置租约时间
	resp, err := s.cli.Grant(context.Background(), lease)
	if err != nil {
		return err
	}
	//注册服务并绑定租约
	_, err = s.cli.Put(context.Background(), s.key, s.val, clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}
	//设置续租 定期发送需求请求
	leaseRespChan, err := s.cli.KeepAlive(context.Background(), resp.ID)

	if err != nil {
		return err
	}
	s.leaseID = resp.ID
	s.logger.Info("lease id", zap.Any("leaseId", s.leaseID))
	s.keepAliveChan = leaseRespChan
	//log.Printf("Put key:%s  val:%s  success!", s.key, s.val)
	s.logger.Info("put key", zap.String("key", s.key), zap.String("value", s.val))
	return nil
}

//ListenLeaseRespChan 监听 续租情况
func (s *ServiceRegister) ListenLeaseRespChan() {
	for leaseKeepResp := range s.keepAliveChan {
		s.logger.Debug("续约成功", zap.Any("resp", leaseKeepResp))
	}
	s.logger.Info("关闭续租", zap.String("key", s.key))
}

// Close 注销服务
func (s *ServiceRegister) Close() error {
	//撤销租约
	if _, err := s.cli.Revoke(context.Background(), s.leaseID); err != nil {
		return err
	}
	s.logger.Info("撤销租约", zap.String("key", s.key))
	return s.cli.Close()
}

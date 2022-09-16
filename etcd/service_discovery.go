package etcd

import (
	"context"
	"go.uber.org/zap"
	"strings"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
)

//ServiceDiscovery 服务发现
type ServiceDiscovery struct {
	cli        *clientv3.Client  //etcd client
	serverList map[string]string //服务列表
	lock       sync.Mutex
	logger     *zap.Logger // 日志
}

//NewServiceDiscovery  新建发现服务
func NewServiceDiscovery(endpoints []string, logger *zap.Logger) *ServiceDiscovery {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		logger.Fatal("new service discovery is error", zap.Error(err))
	}

	return &ServiceDiscovery{
		cli:        cli,
		serverList: make(map[string]string),
		logger:     logger,
	}
}

//WatchService 初始化服务列表和监视
func (s *ServiceDiscovery) WatchService(prefix string) error {
	//根据前缀获取现有的key
	resp, err := s.cli.Get(context.Background(), prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, ev := range resp.Kvs {
		s.SetServiceList(string(ev.Key), string(ev.Value))
	}

	//监视前缀，修改变更的server
	go s.watcher(prefix)
	return nil
}

//watcher 监听前缀
func (s *ServiceDiscovery) watcher(prefix string) {
	rch := s.cli.Watch(context.Background(), prefix, clientv3.WithPrefix())
	s.logger.Info("watching prefix now...", zap.String("prefix", prefix))
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT: //修改或者新增
				s.SetServiceList(string(ev.Kv.Key), string(ev.Kv.Value))
			case mvccpb.DELETE: //删除
				s.DelServiceList(string(ev.Kv.Key))
			}
		}
	}
}

//SetServiceList 新增服务地址
func (s *ServiceDiscovery) SetServiceList(key, val string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.serverList[key] = val
	s.logger.Info("put key", zap.String("key", key), zap.String("value", val))
}

//DelServiceList 删除服务地址
func (s *ServiceDiscovery) DelServiceList(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.serverList, key)
	s.logger.Info("del key:", zap.String("key", key))
}

//GetServices 获取服务地址
func (s *ServiceDiscovery) GetServices() []string {
	s.lock.Lock()
	defer s.lock.Unlock()
	addrs := make([]string, 0)

	for _, v := range s.serverList {
		addrs = append(addrs, v)
	}
	return addrs
}

func (s *ServiceDiscovery) GetService(prefix string) []string {
	s.lock.Lock()
	defer s.lock.Unlock()
	var res []string
	for k, v := range s.serverList {
		if strings.HasPrefix(k, prefix) {
			res = append(res, v)
		}
	}
	return res
}

func (s *ServiceDiscovery) extractAdders(resp *clientv3.GetResponse) []string {
	addrs := make([]string, 0)
	if resp == nil || resp.Kvs == nil {
		return addrs
	}
	for i := range resp.Kvs {
		if v := resp.Kvs[i].Value; v != nil {
			s.SetServiceList(string(resp.Kvs[i].Key), string(resp.Kvs[i].Value))
			addrs = append(addrs, string(v))
		}
	}
	return addrs
}

//Close 关闭服务
func (s *ServiceDiscovery) Close() error {
	return s.cli.Close()
}

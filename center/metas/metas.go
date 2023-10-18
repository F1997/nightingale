package metas

import (
	"context"
	"sync"
	"time"

	"github.com/F1997/nightingale/models"
	"github.com/F1997/nightingale/storage"

	"github.com/toolkits/pkg/logger"
)

type Set struct {
	sync.RWMutex
	items map[string]models.HostMeta
	redis storage.Redis
}

// 创建并初始化 Set， 启动元数据持久化协程
func New(redis storage.Redis) *Set {
	set := &Set{
		items: make(map[string]models.HostMeta),
		redis: redis,
	}

	set.Init()
	return set
}

// 启动元数据持久化协程
func (s *Set) Init() {
	go s.LoopPersist()
}

// 批量设置主机元数据
func (s *Set) MSet(items map[string]models.HostMeta) {
	s.Lock()
	defer s.Unlock()
	for ident, meta := range items {
		s.items[ident] = meta
	}
}

// 设置单个主机的元数据
func (s *Set) Set(ident string, meta models.HostMeta) {
	s.Lock()
	defer s.Unlock()
	s.items[ident] = meta
}

// 每隔1s，调用一次 persist()
func (s *Set) LoopPersist() {
	for {
		time.Sleep(time.Second)
		s.persist()
	}
}

func (s *Set) persist() {
	var items map[string]models.HostMeta

	s.Lock()
	if len(s.items) == 0 {
		s.Unlock()
		return
	}

	items = s.items
	s.items = make(map[string]models.HostMeta)
	s.Unlock()

	s.updateMeta(items)
}

func (s *Set) updateMeta(items map[string]models.HostMeta) {
	m := make(map[string]models.HostMeta, 100)
	num := 0

	for _, meta := range items {
		m[meta.Hostname] = meta
		num++
		if num == 100 {
			if err := s.updateTargets(m); err != nil {
				logger.Errorf("failed to update targets: %v", err)
			}
			m = make(map[string]models.HostMeta, 100)
			num = 0
		}
	}

	if err := s.updateTargets(m); err != nil {
		logger.Errorf("failed to update targets: %v", err)
	}
}

// 将数据持久化到 Redis 中
func (s *Set) updateTargets(m map[string]models.HostMeta) error {
	count := int64(len(m))
	if count == 0 {
		return nil
	}

	newMap := make(map[string]interface{}, count)
	for ident, meta := range m {
		newMap[models.WrapIdent(ident)] = meta
	}
	err := storage.MSet(context.Background(), s.redis, newMap)
	return err
}

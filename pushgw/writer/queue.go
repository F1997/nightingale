package writer

import (
	"container/list"
	"sync"

	"github.com/prometheus/prometheus/prompb"
)

type SafeList struct {
	sync.RWMutex // 读写锁,安全访问列表
	L            *list.List
}

func NewSafeList() *SafeList {
	return &SafeList{L: list.New()}
}

// 将一个元素插入到列表的头部，并返回插入的元素
func (sl *SafeList) PushFront(v interface{}) *list.Element {
	sl.Lock()
	e := sl.L.PushFront(v)
	sl.Unlock()
	return e
}

// 将多个元素批量插入到列表的头部
func (sl *SafeList) PushFrontBatch(vs []interface{}) {
	sl.Lock()
	for _, item := range vs {
		sl.L.PushFront(item)
	}
	sl.Unlock()
}

// 从列表的尾部弹出指定数量的元素，并返回 *prompb.TimeSeries 的切片
func (sl *SafeList) PopBack(max int) []*prompb.TimeSeries {
	sl.Lock()

	count := sl.L.Len()
	if count == 0 {
		sl.Unlock()
		return []*prompb.TimeSeries{}
	}

	if count > max {
		count = max
	}

	items := make([]*prompb.TimeSeries, 0, count)
	for i := 0; i < count; i++ {
		item := sl.L.Remove(sl.L.Back())
		sample, ok := item.(*prompb.TimeSeries)
		if ok {
			items = append(items, sample)
		}
	}

	sl.Unlock()
	return items
}

// 清空列表中的所有元素
func (sl *SafeList) RemoveAll() {
	sl.Lock()
	sl.L.Init()
	sl.Unlock()
}

// 获取列表的长度
func (sl *SafeList) Len() int {
	sl.RLock()
	size := sl.L.Len()
	sl.RUnlock()
	return size
}

// SafeList with Limited Size
type SafeListLimited struct {
	maxSize int       // 列表的最大容量
	SL      *SafeList // 继承 SafeList
}

func NewSafeListLimited(maxSize int) *SafeListLimited {
	return &SafeListLimited{SL: NewSafeList(), maxSize: maxSize}
}

func (sll *SafeListLimited) PopBack(max int) []*prompb.TimeSeries {
	return sll.SL.PopBack(max)
}

func (sll *SafeListLimited) PushFront(v interface{}) bool {
	if sll.SL.Len() >= sll.maxSize {
		return false
	}

	sll.SL.PushFront(v)
	return true
}

func (sll *SafeListLimited) PushFrontBatch(vs []interface{}) bool {
	if sll.SL.Len() >= sll.maxSize {
		return false
	}

	sll.SL.PushFrontBatch(vs)
	return true
}

func (sll *SafeListLimited) RemoveAll() {
	sll.SL.RemoveAll()
}

func (sll *SafeListLimited) Len() int {
	return sll.SL.Len()
}

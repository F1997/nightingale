package dumper

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/time"
)

type SyncRecord struct {
	Timestamp int64
	Mills     int64
	Count     int
	Message   string
}

func (sr *SyncRecord) String() string {
	var sb strings.Builder
	sb.WriteString("timestamp: ")
	sb.WriteString(time.Format(sr.Timestamp))
	sb.WriteString(", mills: ")
	sb.WriteString(fmt.Sprint(sr.Mills, "ms"))
	sb.WriteString(", count: ")
	sb.WriteString(fmt.Sprint(sr.Count))
	sb.WriteString(", message: ")
	sb.WriteString(sr.Message)

	return sb.String()
}

type SyncRecords struct {
	Current *SyncRecord
	Last    *SyncRecord
}

type SyncDumper struct {
	sync.RWMutex
	records map[string]*SyncRecords
}

// 创建一个新的 SyncDumper 实例
func NewSyncDumper() *SyncDumper {
	return &SyncDumper{
		records: make(map[string]*SyncRecords),
	}
}

var syncDumper = NewSyncDumper()

func (sd *SyncDumper) Put(key string, timestamp, mills int64, count int, message string) {
	sr := &SyncRecord{
		Timestamp: timestamp,
		Mills:     mills,
		Count:     count,
		Message:   message,
	}

	sd.Lock()
	defer sd.Unlock()

	if _, ok := sd.records[key]; !ok {
		sd.records[key] = &SyncRecords{Current: sr}
		return
	}

	sd.records[key].Last = sd.records[key].Current
	sd.records[key].Current = sr
}

// busi_groups:
// last: timestamp, mills, count
// curr: timestamp, mills, count
func (sd *SyncDumper) Sprint() string {
	sd.RLock()
	defer sd.RUnlock()

	var sb strings.Builder
	sb.WriteString("\n")

	for k, v := range sd.records {
		sb.WriteString(k)
		sb.WriteString(":\n")
		if v.Last != nil {
			sb.WriteString("last: ")
			sb.WriteString(v.Last.String())
			sb.WriteString("\n")
		}
		sb.WriteString("curr: ")
		sb.WriteString(v.Current.String())
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// 配置路由，提供了一个 /dumper/sync 的 HTTP GET 请求
// 只允许本地访问（127.0.0.1 或 ::1）
func (sd *SyncDumper) ConfigRouter(r *gin.Engine) {
	r.GET("/dumper/sync", func(c *gin.Context) {
		clientIP := c.ClientIP()
		if clientIP != "127.0.0.1" && clientIP != "::1" {
			c.String(403, "forbidden")
			return
		}
		c.String(200, sd.Sprint())
	})
}

// 向 syncDumper 添加同步记录
func PutSyncRecord(key string, timestamp, mills int64, count int, message string) {
	syncDumper.Put(key, timestamp, mills, count, message)
}

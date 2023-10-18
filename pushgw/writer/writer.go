package writer

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/F1997/nightingale/pushgw/pconf"

	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/prometheus/prompb"
	"github.com/toolkits/pkg/concurrent/semaphore"
	"github.com/toolkits/pkg/logger"
)

// 写入目标的配置信息和客户端
type WriterType struct {
	Opts             pconf.WriterOptions // 配置信息，包括 URL、认证信息等。
	ForceUseServerTS bool                // 是否强制使用服务器时间戳
	Client           api.Client          // Prometheus 客户端，用于发送数据到远程目标
}

// 根据配置的重写规则对时间序列数据进行标签重写，返回经过重写后的时间序列数据
func (w WriterType) writeRelabel(items []*prompb.TimeSeries) []*prompb.TimeSeries {
	ritems := make([]*prompb.TimeSeries, 0, len(items))
	for _, item := range items {
		lbls := Process(item.Labels, w.Opts.WriteRelabels...)
		if len(lbls) == 0 {
			continue
		}
		ritems = append(ritems, item)
	}
	return ritems
}

// 将时间序列数据写入远程目标，根据配置进行数据的重写和请求的发送
func (w WriterType) Write(items []*prompb.TimeSeries, sema *semaphore.Semaphore, headers ...map[string]string) {
	defer sema.Release()
	if len(items) == 0 {
		return
	}

	// 对时间序列数据进行标签重写
	items = w.writeRelabel(items)
	if len(items) == 0 {
		return
	}

	if w.ForceUseServerTS {
		ts := time.Now().UnixMilli()
		for i := 0; i < len(items); i++ {
			if len(items[i].Samples) == 0 {
				continue
			}
			items[i].Samples[0].Timestamp = ts
		}
	}

	// 时间序列数据添加到 prompb.WriteRequest
	req := &prompb.WriteRequest{
		Timeseries: items,
	}

	// snappy 压缩数据
	data, err := proto.Marshal(req)
	if err != nil {
		logger.Warningf("marshal prom data to proto got error: %v, data: %+v", err, items)
		return
	}

	// 发送 HTTP POST 请求到目标 URL
	if err := w.Post(snappy.Encode(nil, data), headers...); err != nil {
		logger.Warningf("post to %s got error: %v", w.Opts.Url, err)
		logger.Warning("example timeseries:", items[0].String())
	}
}

// 发送 HTTP POST 请求到远程目标
func (w WriterType) Post(req []byte, headers ...map[string]string) error {
	httpReq, err := http.NewRequest("POST", w.Opts.Url, bytes.NewReader(req))
	if err != nil {
		logger.Warningf("create remote write request got error: %s", err.Error())
		return err
	}

	// 设置请求头信息
	httpReq.Header.Add("Content-Encoding", "snappy")                 // 请求体使用 snappy 压缩
	httpReq.Header.Set("Content-Type", "application/x-protobuf")     // 请求体的内容类型为 Protocol Buffers 格式
	httpReq.Header.Set("User-Agent", "n9e")                          // 用户代理信息
	httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0") // Prometheus 远程写入的版本信息

	if len(headers) > 0 {
		for k, v := range headers[0] {
			httpReq.Header.Set(k, v)
		}
	}

	if w.Opts.BasicAuthUser != "" {
		httpReq.SetBasicAuth(w.Opts.BasicAuthUser, w.Opts.BasicAuthPass) // 使用 httpReq.SetBasicAuth 方法设置 Basic 认证
	}

	headerCount := len(w.Opts.Headers)
	if headerCount > 0 && headerCount%2 == 0 {
		for i := 0; i < len(w.Opts.Headers); i += 2 {
			httpReq.Header.Add(w.Opts.Headers[i], w.Opts.Headers[i+1])
			if w.Opts.Headers[i] == "Host" {
				httpReq.Host = w.Opts.Headers[i+1]
			}
		}
	}

	// 使用 Prometheus 客户端的 Do 方法发送 HTTP 请求，获取响应信息 resp 和响应体内容 body。
	resp, body, err := w.Client.Do(context.Background(), httpReq)
	if err != nil {
		logger.Warningf("push data with remote write request got error: %v, response body: %s", err, string(body))
		return err
	}

	if resp.StatusCode >= 400 {
		err = fmt.Errorf("push data with remote write request got status code: %v, response body: %s", resp.StatusCode, string(body))
		return err
	}

	return nil
}

// 多个时间序列数据写入目标的集合
type WritersType struct {
	pushgw   pconf.Pushgw
	backends map[string]WriterType
	queues   map[string]*IdentQueue
	sema     *semaphore.Semaphore
	sync.RWMutex
}

// 标识队列
type IdentQueue struct {
	list    *SafeListLimited
	closeCh chan struct{}
	ts      int64
}

func NewWriters(pushgwConfig pconf.Pushgw) *WritersType {
	writers := &WritersType{
		backends: make(map[string]WriterType),
		queues:   make(map[string]*IdentQueue),
		pushgw:   pushgwConfig,
		sema:     semaphore.NewSemaphore(pushgwConfig.WriteConcurrency),
	}

	writers.Init()
	go writers.CleanExpQueue()
	return writers
}

func (ws *WritersType) Put(name string, writer WriterType) {
	ws.backends[name] = writer
}

// 清理过期的标识队列
func (ws *WritersType) CleanExpQueue() {
	for {
		// 获取 WritersType 实例的互斥锁
		ws.Lock()
		// 遍历处理所有标识符
		for ident := range ws.queues {
			identQueue := ws.queues[ident]
			if identQueue == nil {
				delete(ws.queues, ident)
				logger.Warningf("Write channel(%s) not found", ident)
				continue
			}

			if time.Now().Unix()-identQueue.ts > 3600 {
				close(identQueue.closeCh)
				delete(ws.queues, ident)
			}
		}
		ws.Unlock()
		time.Sleep(time.Second * 600)
	}
}

// 将时间序列数据推送到指定的目标队列
func (ws *WritersType) PushSample(ident string, v interface{}) {
	ws.RLock()
	identQueue := ws.queues[ident]
	ws.RUnlock()
	if identQueue == nil {
		identQueue = &IdentQueue{
			list:    NewSafeListLimited(ws.pushgw.WriterOpt.QueueMaxSize),
			closeCh: make(chan struct{}),
			ts:      time.Now().Unix(),
		}

		ws.Lock()
		ws.queues[ident] = identQueue
		ws.Unlock()
		// 开启一个协程，用于从队列中取出数据并发送到目标
		go ws.StartConsumer(identQueue)
	}

	identQueue.ts = time.Now().Unix()
	succ := identQueue.list.PushFront(v)
	if !succ {
		logger.Warningf("Write channel(%s) full, current channel size: %d", ident, identQueue.list.Len())
	}
}

// 消费队列
func (ws *WritersType) StartConsumer(identQueue *IdentQueue) {
	for {
		select {
		case <-identQueue.closeCh: // 监听 identQueue.closeCh 通道的关闭事件
			logger.Infof("write queue:%v closed", identQueue)
			return
		default:
			series := identQueue.list.PopBack(ws.pushgw.WriterOpt.QueuePopSize)
			if len(series) == 0 {
				time.Sleep(time.Millisecond * 400)
				continue
			}
			for key := range ws.backends {
				ws.sema.Acquire()
				go ws.backends[key].Write(series, ws.sema)
			}
		}
	}
}

// 初始化时间序列数据写入目标，包括创建 Prometheus 客户端、设置 HTTP Transport 选项等
func (ws *WritersType) Init() error {
	opts := ws.pushgw.Writers

	for i := 0; i < len(opts); i++ {
		tlsConf, err := opts[i].ClientConfig.TLSConfig()
		if err != nil {
			return err
		}

		trans := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   time.Duration(opts[i].DialTimeout) * time.Millisecond,
				KeepAlive: time.Duration(opts[i].KeepAlive) * time.Millisecond,
			}).DialContext,
			ResponseHeaderTimeout: time.Duration(opts[i].Timeout) * time.Millisecond,
			TLSHandshakeTimeout:   time.Duration(opts[i].TLSHandshakeTimeout) * time.Millisecond,
			ExpectContinueTimeout: time.Duration(opts[i].ExpectContinueTimeout) * time.Millisecond,
			MaxConnsPerHost:       opts[i].MaxConnsPerHost,
			MaxIdleConns:          opts[i].MaxIdleConns,
			MaxIdleConnsPerHost:   opts[i].MaxIdleConnsPerHost,
			IdleConnTimeout:       time.Duration(opts[i].IdleConnTimeout) * time.Millisecond,
		}

		if tlsConf != nil {
			trans.TLSClientConfig = tlsConf
		}

		cli, err := api.NewClient(api.Config{
			Address:      opts[i].Url,
			RoundTripper: trans,
		})

		if err != nil {
			return err
		}

		writer := WriterType{
			Opts:             opts[i],
			Client:           cli,
			ForceUseServerTS: ws.pushgw.ForceUseServerTS,
		}

		ws.Put(opts[i].Url, writer)
	}

	return nil
}

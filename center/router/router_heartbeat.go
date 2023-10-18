package router

import (
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"strings"
	"time"

	"github.com/F1997/nightingale/models"

	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
	"github.com/toolkits/pkg/logger"
)

func (rt *Router) heartbeat(c *gin.Context) {
	var bs []byte
	var err error
	var r *gzip.Reader
	var req models.HostMeta

	if c.GetHeader("Content-Encoding") == "gzip" {
		// gzip 解压缩
		r, err = gzip.NewReader(c.Request.Body)
		if err != nil {
			c.String(400, err.Error())
			return
		}
		defer r.Close()
		bs, err = ioutil.ReadAll(r)
		ginx.Dangerous(err)
	} else {
		defer c.Request.Body.Close()
		bs, err = ioutil.ReadAll(c.Request.Body)
		ginx.Dangerous(err)
	}

	err = json.Unmarshal(bs, &req)
	ginx.Dangerous(err)

	// maybe from pushgw
	if req.Offset == 0 {
		// req.Offset 设置为当前时间戳减去 UnixTime 字段的值，以计算时间偏移
		req.Offset = (time.Now().UnixMilli() - req.UnixTime)
	}

	if req.RemoteAddr == "" {
		// req.RemoteAddr 设置为客户端的 IP 地址
		req.RemoteAddr = c.ClientIP()
	}

	// 存储主机元数据
	rt.MetaSet.Set(req.Hostname, req)
	var items = make(map[string]struct{})
	items[req.Hostname] = struct{}{}
	rt.IdentSet.MSet(items)

	// 判断机器缓存中是否存在与主机名匹配的目标信息
	if target, has := rt.TargetCache.Get(req.Hostname); has && target != nil {
		var defGid int64 = -1
		// 请求中的 gid 和主机 IP 地址是否与目标信息匹配
		gid := ginx.QueryInt64(c, "gid", defGid)
		hostIpStr := strings.TrimSpace(req.HostIp)
		if gid == defGid { //set gid value from cache
			gid = target.GroupId
		}
		logger.Debugf("heartbeat gid: %v, host_ip: '%v', target: %v", gid, hostIpStr, *target)
		if gid != target.GroupId || hostIpStr != target.HostIp { // if either gid or host_ip has a new value
			// 更新目标信息的主机 IP 地址和 gid
			err = models.TargetUpdateHostIpAndBgid(rt.Ctx, req.Hostname, hostIpStr, gid)
		}
	}

	ginx.NewRender(c).Message(err)
}

package router

import (
	"github.com/F1997/nightingale/pushgw/idents"
	"github.com/gin-gonic/gin"
	"github.com/toolkits/pkg/ginx"
)

func (rt *Router) targetUpdate(c *gin.Context) {
	var f idents.TargetUpdate
	// 从请求中读取 JSON 数据并将其解析为 idents.TargetUpdate 结构
	ginx.BindJSON(c, &f)
	// 进行更新，并返回更新的结果。
	ginx.NewRender(c).Message(rt.IdentSet.UpdateTargets(f.Lst, f.Now))
}

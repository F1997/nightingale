# nightingale 目录&文件

## nightingale/alert 告警模块
- 告警相关配置 
    > nightingale/alert/aconf/conf.go
- 指标监控的 Prometheus 指标（metrics）的创建和注册
    > nightingale/alert/astats/stats.go 
- 处理异常点（AnomalyPoint）以及将 Prometheus 指标（metrics）转换为异常点的功能
    > nightingale/alert/common/conv.go 
- 一些通用的函数，用于处理规则（rule）和标签（tag）匹配等相关逻辑 
    > nightingale/alert/common/key.go
- 定义了一个名为 Consumer 的结构体和相关方法，用于实现消费者的逻辑，主要是处理告警事件的消费和持久化操作
    > nightingale/alert/dispatch/consume.go 
- 实现告警通知的自动化处理
    > nightingale/alert/dispatch/dispatch.go
- 定义了一个用于记录告警事件日志的函数 LogEvent
    > nightingale/alert/dispatch/log.go
- 定义了一个名为 NotifyChannels 的自定义类型，它是一个字符串到布尔值的映射，用于表示通知渠道的订阅状态。通常，每个通道都对应一个布尔值，用于表示是否订阅了该通道
    > nightingale/alert/dispatch/notify_channel.go
- 定义了一个名为 NotifyTarget 的结构体，用于维护需要发送通知的目标信息，包括用户-通道/回调/钩子信息；定义了一些用于实现告警事件到信息接收者的路由策略的函数；通过操作 NotifyTarget 实例来构建信息接收者的目标，从而实现了不同的路由策略
    > nightingale/alert/dispatch/notify_target.go
- 定义了一个名为 Scheduler 的结构体，它是告警规则调度器，负责同步和管理告警规则的执行
    > nightingale/alert/eval/alert_rule.go
- AlertRuleWorker 主要负责执行告警规则的评估逻辑，根据不同类型的规则调用不同的方法获取评估数据，并将数据传递给告警规则处理器进行后续处理
    > nightingale/alert/eval/eval.go
- 定义了一些用于判断是否要对告警进行屏蔽（mute）的策略函数
    > nightingale/alert/mute/mute.go
- 实现了一种一致性哈希环（Consistent Hash Ring）的数据结构和管理方法，用于分片和负载均衡的场景。
    > nightingale/alert/naming/hashring.go
- 实现告警引擎的命名和发现功能，确保告警引擎实例能够注册到中心节点并与数据源建立对应关系，同时根据活跃服务器列表进行负载均衡，确保告警规则的分片和并行处理。定期清理不活跃的告警引擎实例记录，以确保数据的准确性和可用性。
    > nightingale/alert/naming/heartbeat.go
- 提供了对当前告警事件数据的安全访问和管理，确保多个并发操作时不会出现数据竞争和不一致的情况。
    > nightingale/alert/process/alert_cur_event.go
- 告警事件的处理器 (Processor)，用于处理来自监控系统的异常事件，根据预定义的告警规则进行处理和通知。
    > nightingale/alert/process/process.go
- 实现了一个事件队列 (EventQueue)，用于存储待处理的告警事件
    > nightingale/alert/queue/queue.go
- 定义了一个 RecordRuleContext 结构体，用于处理录制规则（Recording Rule）的相关逻辑
    > nightingale/alert/record/prom_rule.go
- 将 Prometheus 查询结果转换为用于推送到数据存储的时间序列数据格式，同时保留了指标的标签信息
    > nightingale/alert/record/sample.go
- 实现了Recording Rule的调度和管理功能，定期同步规则并启动或停止相应的规则执行
    > nightingale/alert/record/scheduler.go
- HTTP请求处理程序，用于将事件推送到Nightangle的告警队列（queue.EventQueue）中
    > nightingale/alert/router/router_event.go
- 定义了Nightangle的HTTP路由处理程序，用于处理与告警相关的HTTP请求
    > nightingale/alert/router/router.go
- 处理告警事件的回调发送操作，主要是发送告警事件到指定的URL或Ibex任务
    > nightingale/alert/sender/callback.go
- 发送DingTalk消息
    > nightingale/alert/sender/dingtalk.go
- 发送邮件通知
    > nightingale/alert/sender/email.go
- 发送飞书消息
    > nightingale/alert/sender/feishu.go
- 发送飞书卡片消息
    > nightingale/alert/sender/feishucard.go
- 发送Mattermost消息
    > nightingale/alert/sender/mm.go
<!-- -   
    > nightingale/alert/sender/plugin_cmd_unix.go
-   
    > nightingale/alert/sender/plugin_cmd_windows.go -->
- 执行通知脚本
    > nightingale/alert/sender/plugin.go
- 消息通知发送的通用接口和相关的实现
    > nightingale/alert/sender/sender.go
- Telegram 消息通知发送器，用于向 Telegram 机器人发送消息通知
    > nightingale/alert/sender/telegram.go
- 发送 Webhooks 通知，它遍历一个 Webhooks 列表并发送通知
    > nightingale/alert/sender/webhook.go
- 发送企业微信（WeCom）的Markdown消息通知
    > nightingale/alert/sender/wecom.go
- 初始化和启动Nightangle的Alerting模块，包括了缓存、调度器、消费者等组件的初始化和启动，以及后台任务的启动
    > nightingale/alert/alert.go

## nightingale/center 中心服务
- 定义了用于存储配置信息的结构体，并提供了一个方法用于在加载配置之前进行预检查。配置信息包括插件配置、文件路径、国际化请求头、指标描述信息等
    > nightingale/center/cconf/conf.go
- 事件的样例 JSON 数据
    > nightingale/center/cconf/event_example.go
- 定义用于加载指标描述信息的函数和相关变量
    > nightingale/center/cconf/metric.go
- 定义用于加载操作信息的函数和相关变量
    > nightingale/center/cconf/ops.go
- 定义一个全局变量 Plugins，用于存储一组插件信息
    > nightingale/center/cconf/plugin.go
- 使用 Prometheus 来监控 Go 服务的性能和运行情况，可以用于实时监控服务的响应时间、请求总数以及服务的正常运行时间等信息
    > nightingale/center/cstats/stats.go
- 元数据管理的功能，将主机元数据存储在内存中，定期将内存中的数据持久化到 Redis 中，以确保数据的持久性和可用性。
    > nightingale/center/metas/metas.go
- 路由文件
    > nightingale/center/router/router
- 实现管理告警聚合视图功能
    > nightingale/center/router/router_alert_aggr_view.go
- 处理有关告警事件的请求
    > nightingale/center/router/router_alert_cur_event.go
- 处理有关历史告警的请求
    > nightingale/center/router/router_alert_his_event.go
- 处理与告警规则相关的请求
    > nightingale/center/router/router_alert_rule.go
- 处理与告警订阅相关的请求
    > nightingale/center/router/router_alert_subscribe.go
- 管理和操作仪表板
    > nightingale/center/router/router_board.go
- 处理内置仪表板和告警规则
    > nightingale/center/router/router_builtin.go
- 处理业务组相关请求
    > nightingale/center/router/router_busi_group.go
- 生成&验证验证码，Reids 存储验证码
    > nightingale/center/router/router_captcha.go
- 实现图表分享相关的功能
    > nightingale/center/router/router_chart_share.go
- 实现获取通知渠道和联系人密钥的功能，管理通知渠道和联系人信息
    > nightingale/center/router/router_config.go
- 管理配置项功能
    > nightingale/center/router/router_configs.go
- 加密和解密
    > nightingale/center/router/router_crypto.go
- 定义了一些结构体，用于表示仪表板、图表组和图表的数据模型
    > nightingale/center/router/router_dashboard.go
- 处理数据源相关操作
    > nightingale/center/router/router_datasource.go
- 用于处理与 ElasticSearch（ES）Index Pattern 相关的操作
    > nightingale/center/router/router_es_index_pattern.go
- 处理函数执行各种操作，包括统计数据、查询数据源的ID、创建任务、验证表单、获取特定的数据对象等
    > nightingale/center/router/router_funcs.go
- 用于处理来自主机的心跳请求，更新主机的元数据信息以及可能更新目标信息的主机 IP 地址和 gid
    > nightingale/center/router/router_heartbeat.go
- 用户登录、登出、刷新令牌、以及单点登录（SSO）相关的路由功能，其中包括了不同的登录方式，如用户名密码登录、CAS 登录、OIDC 登录、OAuth2 登录，以及相关的配置更新和获取等功能。
    > nightingale/center/router/router_login.go
- 指标描述
    > nightingale/center/router/router_metric_desc.go
- 指标视图
    > nightingale/center/router/router_metric_view.go
- 实现了告警屏蔽（Alert Mute）相关的路由功能
    > nightingale/center/router/router_mute.go
- 处理身份验证和授权
    > nightingale/center/router/router_mw.go
- 通知配置
    > nightingale/center/router/router_notify_config.go
- 通知模板的管理和操作
    > nightingale/center/router/router_notify_tpl.go
- 用于反向代理的HTTP请求处理函数，用于将来自客户端的HTTP请求转发到其他HTTP服务的功能
    > nightingale/center/router/router_proxy.go
<!-- - 
    > nightingale/center/router/router_recording_rule.go -->
- 处理角色和操作之间的关联关系，以及获取操作的信息
    > nightingale/center/router/router_role_operation.go
- 管理角色和权限的信息，包括创建、更新、删除角色，获取角色列表，以及获取当前用户的权限列表等操作
    > nightingale/center/router/router_role.go
- 用于处理与用户个人信息、个人资料和密码管理相关的请求
    > nightingale/center/router/router_self.go
- 用于处理与告警引擎服务器相关的请求，包括获取服务器列表、集群列表、更新心跳信息和获取活跃服务器列表等操作。
    > nightingale/center/router/router_server.go
- 管理和监控目标主机的信息，包括获取主机列表、绑定/解绑标签、更新备注信息、更新业务组ID等操作
    > nightingale/center/router/router_target.go
- 管理自愈脚本模板，包括创建、更新、删除、查询自愈脚本模板以及绑定/解绑标签等操作
    > nightingale/center/router/router_task_tpl.go
- 用于告警自愈的创建、查询、记录管理等操作，以及与远程IBEX服务的通信
    > nightingale/center/router/router_task.go
- 主要用于用户组（团队）的创建、查询、更新、删除以及成员管理等操作
    > nightingale/center/router/router_user_group.go
- 用于用户的创建、查询、更新、删除以及个人资料和密码管理等操作
    > nightingale/center/router/router_user.go
- 定义了一个名为 SsoClient 的结构体，用于存储不同类型的单点登录（SSO）客户端配置，包括 OIDC、LDAP、CAS 和 OAuth2。
    > nightingale/center/sso/init.go
- 初始化函数，它的主要功能是初始化应用程序的各个组件、配置和路由。
    > nightingale/center/center.go

## nightingale/cmd 启动
- 初始化和运行NightinGale Alert服务
    > nightingale/cmd/alert/main.go
- 初始化和运行NightinGale Center服务
    > nightingale/cmd/center/main.go
- 用于执行Nightingale的数据库升级操作或显示版本信息
    > nightingale/cmd/cli/main.go
<!-- - 初始化Nightingale的告警模块和推送网关模块
    > nightingale/cmd/edge/edge.go
- 初始化应用程序的各个模块、处理系统信号以及执行清理操作
    > nightingale/cmd/edge/main.go -->
- 初始化推送网关模块、处理系统信号以及执行清理操作
    > nightingale/cmd/pushgw/main.go

## nightingale/conf 加载配置
- 加载Nightangle应用程序的配置，并对配置进行预检查和解密
    > nightingale/conf/conf.go
- 用于解密Nightangle配置中的敏感信息
    > nightingale/conf/crypto.go

## nightingale/doc 文档

## nightingale/docker docker 配置

## nightingale/dumper
- 记录和展示同步记录，通过 HTTP 接口 /dumper/sync 可以查看同步记录的状态
    > nightingale/dumper/sync.go

## nightingale/etc 项目配置

## nightingale/front

## nightingale/integrations 第三方插件

## nightingale/memsto 缓存模块
- 定义了一个缓存告警屏蔽数据的类型，并提供了与该缓存相关的操作方法
    > nightingale/memsto/alert_mute_cache.go
- 管理和维护告警规则数据的缓存，并定时同步最新的数据到缓存
    > nightingale/memsto/alert_rule_cache.go
- 维护和同步告警订阅数据的缓存
    > nightingale/memsto/alert_subscribe_cache.go
- 维护和同步业务组数据的缓存
    > nightingale/memsto/busi_group_cache.go
- 维护和同步数据源信息的缓存
    > nightingale/memsto/datasource_cache.go
<!-- - exit 时执行
    > nightingale/memsto/memsto.go -->
- 在程序启动时缓存加载配置信息并定期同步
    > nightingale/memsto/notify_config.go
- 
    > nightingale/memsto/recording_rule_cache.go
- 
    > nightingale/memsto/stat.go
- 
    > nightingale/memsto/target_cache.go
- 
    > nightingale/memsto/user_cache.go
- 
    > nightingale/memsto/user_group_cache.go

## nightingale/models 数据库模型

## nightingale/pkg 工具库

## nightingale/prom Prometheus 模块

## nightingale/pushgw 推送网关
- 更新 idents 的时间戳信息到目标存储
    > nightingale/pushgw/idents/idents.go
- 配置Pushgateway的行为和选项
    > nightingale/pushgw/pconf/conf.go
- 根据目标信息，为时间序列添加标签，包括目标标签和业务组标签，并根据配置项决定是否覆盖已有的标签值。
    > nightingale/pushgw/router/fns.go
- 处理Datadog格式的时间序列数据推送，并将其转换为Prometheus格式，然后根据标识符或指标名称将数据转发，同时记录数据处理的统计信息
    > nightingale/pushgw/router/router_datadog.go
- 接收OpenFalcon格式的指标数据，并将其转换为Prometheus格式，然后将数据转发，同时记录数据处理的统计信息。
    > nightingale/pushgw/router/router_openfalcon.go
- 接收OpenTSDB格式的指标数据，将其转换为Prometheus格式，并将数据转发，同时记录数据处理的统计信息
    > nightingale/pushgw/router/router_opentsdb.go
- 处理心跳请求
    > nightingale/pushgw/router/router_heartbeat.go
- 处理Prometheus Remote Write API请求的路由处理函数，它负责将请求体中的时间序列数据解码并处理，包括提取标识符、数据转发和统计信息记录
    > nightingale/pushgw/router/router_remotewrite.go
- 接收和处理目标标识符的更新请求，然后将更新结果以 JSON 格式返回给客户端
    > nightingale/pushgw/router/router_target.go
- 定义了一个 Prometheus 指标（metric） CounterSampleTotal，用于统计接收的样本数量
    > nightingale/pushgw/router/stat.go
- 用于处理数据的接收和路由
    > nightingale/pushgw/router/router.go
- 存储和操作 Prometheus 时间序列数据的队列
    > nightingale/pushgw/writer/queue.go
- 处理Prometheus时间序列数据的标签
    > nightingale/pushgw/writer/relabel.go
- 将时间序列数据按照配置信息发送到不同的远程目标，并根据需要进行数据处理和重写
    > nightingale/pushgw/writer/writer.go
- 推送网关的初始化代码，主要功能包括配置初始化、日志初始化、身份标识管理、统计信息初始化、缓存初始化、写入器初始化以及路由配置等。
    > nightingale/pushgw/pushgw.go

## nightingale/storage Redis
- 与 Redis 数据存储进行交互
    > nightingale/storage/redis.go
- 创建和初始化一个 gorm.DB 数据库连接对象
    > nightingale/storage/storage.go


## 数据存储
- 采集器:采集到数据 -> 请求 n9e-pushgw -> /prometheus/v1/write 写入 prometheus
    - mysql 机器列表
    - redis 心跳
    - 时序库 数据


- /prometheus/v1/write 请求完整链路，向下执行
    - pushgw/router/router_remotewrite.go 
        > remoteWrite()
    - pushgw/router/fns.go 
        > ForwardByIdent()|| ForwardByMetric() 
    - pushgw/writer/writer.go 
        > PushSample() 
        > StartConsumer()
        > ws.backends[].Write() 
        > Post()
        > w.Client.Do()
- /opentsdb/put 请求完整链路，向下执行
    - pushgw/router/router_opentsdb.go
        > openTSDBPut() 
        > Clean()
        > ToProm()
    - pushgw/router/fns.go 
        > ForwardByIdent()|| ForwardByMetric() 
    - pushgw/writer/writer.go 
        > PushSample() 
        > StartConsumer()
        > ws.backends[].Write() 
        > Post()
        > w.Client.Do()
- /openfalcon/push 请求完整链路，向下执行
    - pushgw/router/router_openfalcon.go.go
        > falconPush() 
        > Clean()
        > ToProm()
    - pushgw/router/fns.go 
        > ForwardByIdent()|| ForwardByMetric() 
    - pushgw/writer/writer.go 
        > PushSample() 
        > StartConsumer()
        > ws.backends[].Write() 
        > Post()
        > w.Client.Do()


# 中心启动
- 启动
    > nightingale/cmd/center/main.go
    - 初始化
        > nightingale/center/center.go Initialize()

# 推送网关启动
- 启动
    > nightingale/cmd/center/main.go
    - 初始化
        > nightingale/center/center.go Initialize()

# mongodb

mongodb 监控采集插件，由 [mongodb-exporter](https://github.com/percona/mongodb_exporter)封装而来。

## Configuration

配置文件示例：

```toml
[[instances]]
# log level, enum: panic, fatal, error, warn, warning, info, debug, trace, defaults to info.
log_level = "info"
# append some const labels to metrics
# NOTICE! the instance label is required for dashboards
labels = { instance="mongo-cluster-01" }

# mongodb dsn, see https://www.mongodb.com/docs/manual/reference/connection-string/
# mongodb_uri = "mongodb://127.0.0.1:27017"
mongodb_uri = ""
# if you don't specify the username or password in the mongodb_uri, you can set here. 
# This will overwrite the dsn, it would be helpful when special characters existing in the username or password and you don't want to encode them.
# NOTICE! this user must be granted enough rights to query needed stats, see ../inputs/mongodb/README.md
username = "username@Bj"
password = "password@Bj"
# if set to true, use the direct connection way
# direct_connect = true

# collect all means you collect all the metrics, if set, all below enable_xxx flags in this section will be ignored
collect_all = true
# if set to true, collect databases metrics
# enable_db_stats = true
# if set to true, collect getDiagnosticData metrics
# enable_diagnostic_data = true
# if set to true, collect replSetGetStatus metrics
# enable_replicaset_status = true
# if set to true, collect top metrics by admin command
# enable_top_metrics = true
# if set to true, collect index metrics. You should specify one of the coll_stats_namespaces and the discovering_mode flags.
# enable_index_stats = true
# if set to true, collect collections metrics. You should specify one of the coll_stats_namespaces and the discovering_mode flags.
# enable_coll_stats = true

# Only get stats for the collections matching this list of namespaces. if none set, discovering_mode will be enabled.
# Example: db1.col1,db.col1
# coll_stats_namespaces = []
# Only get stats for index with the collections matching this list of namespaces.
# Example: db1.col1,db.col1
# index_stats_collections = []
# if set to true, replace -1 to DESC for label key_name of the descending_index metrics
# enable_override_descending_index = true

# which exposes metrics with 0.1x compatible metric names has been implemented which simplifies migration from the old version to the current version.
# compatible_mode = true


# [[instances]]
# # interval = global.interval * interval_times
# interval_times = 1

# log_level = "error"

# append some labels to metrics
# labels = { instance="mongo-cluster-02" }
# mongodb_uri = "mongodb://username:password@127.0.0.1:27017"
# collect_all = true
# compatible_mode = true
```

categraf 作为一个 client 连接 MongoDB，需要有足够的权限来收集指标，具体的权限配置请参考[官方文档](https://www.mongodb.com/docs/manual/reference/built-in-roles/#mongodb-authrole-clusterMonitor)。至少具有以下权限才可以：

```json
{
    "role":"clusterMonitor",
    "db":"admin"
},
{
    "role":"read",
    "db":"local"
}
```

授权操作样例：

```shell
mongo -h xxx -u xxx -p xxx --authenticationDatabase admin
> use admin
> db.createUser({user:"categraf",pwd:"categraf",roles: [{role:"read",db:"local"},{"role":"clusterMonitor","db":"admin"}]})
```

## 监控大盘和告警规则

夜莺内置了 MongoDB 的告警规则和监控大盘，克隆到自己的业务组使用即可。虽然文件后缀是 `_exporter` 也可以使用，因为 categraf 这个插件是基于 mongodb-exporter 封装的。

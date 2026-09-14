module bokeoncall

go 1.25.0

// 依赖由 go mod tidy 按 import 自动维护，当前已引入的都是基础设施（与业务无关）：
//   github.com/spf13/viper                              配置加载（internal/conf）
//   go.uber.org/zap + gopkg.in/natefinch/lumberjack.v2   日志与滚动（pkg/logx）
//   gorm.io/gorm + gorm.io/driver/mysql                  MySQL 工单库（internal/infra/mysql）
//   github.com/elastic/go-elasticsearch/v8               知识库检索（internal/infra/es）
//   github.com/minio/minio-go/v7                         对象存储 / 多模态文件（internal/infra/oss）
//
// 写业务时按需再引入（不要提前加）：
//   github.com/gin-gonic/gin                      HTTP 框架（internal/api；也可以继续用标准库 net/http）
//   github.com/apache/rocketmq-clients/golang/v5  RocketMQ 5.x gRPC 客户端（走 broker 8081 proxy）

require (
	github.com/elastic/go-elasticsearch/v8 v8.19.7
	github.com/larksuite/oapi-sdk-go/v3 v3.12.0
	github.com/minio/minio-go/v7 v7.3.0
	github.com/spf13/viper v1.21.0
	go.uber.org/zap v1.28.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gorm.io/driver/mysql v1.6.0
	gorm.io/gorm v1.31.2
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.9.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/klauspost/cpuid/v2 v2.4.0 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/pelletier/go-toml/v2 v2.3.1 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.opentelemetry.io/otel v1.29.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.opentelemetry.io/otel/trace v1.29.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	gopkg.in/ini.v1 v1.67.3 // indirect
)

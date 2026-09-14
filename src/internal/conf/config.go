// Package conf 配置层：只负责把配置读进来并校验，不含任何业务逻辑。
//
// 企业协作约定：
//   - 仓库只提交 configs/config.example.yaml 模板，真实的 configs/config.yaml 不入库（见根目录 .gitignore）；
//   - 每个开发者复制模板生成自己的 config.yaml，个人差异放 configs/config.local.yaml（优先级更高）；
//   - 生产 / CI 不落文件，直接用环境变量注入：前缀 BOKEONCALL_，层级用下划线连接。
//
// 加载顺序（后面的覆盖前面的）：
//
//	内置默认值 < configs/config.yaml < configs/config.local.yaml < 环境变量
//
// 也可以用 -config 指定文件，或用环境变量 BOKEONCALL_CONFIG 指定路径（这两种情况不再搜索默认路径）。
package conf

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// 本地开发兜底凭据：与 scripts/.env.example 里中间件的默认值保持一致。
// 只有在 app.env=local 且配置里留空时才会用到；非 local 环境必须显式配置，见 Validate。
const (
	localMySQLPassword  = "bokeoncall"
	localMinIOAccessKey = "bokeoncall"
	localMinIOSecretKey = "bokeoncall123"
)

// Config 根配置。
type Config struct {
	// LoadedFrom 实际使用的配置文件路径；为空表示没找到文件、使用内置默认值。
	// mapstructure:"-" 表示不从配置文件读取，仅运行时记录，便于启动日志里提示。
	LoadedFrom string `mapstructure:"-"`

	App      AppConfig      `mapstructure:"app"`
	Log      LogConfig      `mapstructure:"log"`
	MySQL    MySQLConfig    `mapstructure:"mysql"`
	Lark     LarkConfig     `mapstructure:"lark"`
	ES       ESConfig       `mapstructure:"es"`
	MinIO    MinIOConfig    `mapstructure:"minio"`
	RocketMQ RocketMQConfig `mapstructure:"rocketmq"`
	Precheck PrecheckConfig `mapstructure:"precheck"`
	SLA      SLAConfig      `mapstructure:"sla"`
}

// AppConfig 应用基础配置。
type AppConfig struct {
	Env         string `mapstructure:"env"` // local / test / prod
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	GroupPrefix string `mapstructure:"group_prefix"`
}

// Address 返回 HTTP 监听地址。
func (c AppConfig) Address() string { return fmt.Sprintf("%s:%d", c.Host, c.Port) }

// LogConfig 日志配置。
type LogConfig struct {
	Level   string `mapstructure:"level"`
	Dir     string `mapstructure:"dir"`
	Console bool   `mapstructure:"console"`
}

// MySQLConfig 业务库配置（对应 scripts/ 里部署的 bokeoncall-mysql）。
type MySQLConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	Charset      string `mapstructure:"charset"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	AutoMigrate  bool   `mapstructure:"auto_migrate"`
}

// DSN 供 gorm mysql driver 使用。
func (c MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.User, c.Password, c.Host, c.Port, c.Database, c.Charset)
}

// LarkConfig 飞书应用配置。
type LarkConfig struct {
	AppID             string `mapstructure:"app_id"`
	AppSecret         string `mapstructure:"app_secret"`
	VerificationToken string `mapstructure:"verification_token"`
	EncryptKey        string `mapstructure:"encrypt_key"`
	BotOpenID         string `mapstructure:"bot_open_id"`
	EventMode         string `mapstructure:"event_mode"`
	APIBase           string `mapstructure:"api_base"`
	TimeoutSeconds    int    `mapstructure:"timeout_seconds"`
}

// Enabled 是否配置了可用的飞书应用凭证。
func (c LarkConfig) Enabled() bool { return c.AppID != "" && c.AppSecret != "" }

// 事件接入方式。
const (
	EventModeCallback = "callback" // HTTP 回调：生产用，需要在飞书开放平台配置回调地址
	EventModeLongConn = "longconn" // 长连接：本地调试用，不需要公网入口
)

// IsLongConn 是否使用长连接接入。
func (c LarkConfig) IsLongConn() bool { return c.EventMode == EventModeLongConn }

// ESConfig Elasticsearch 配置（知识库检索）。
type ESConfig struct {
	Addresses             []string `mapstructure:"addresses"`
	Username              string   `mapstructure:"username"`
	Password              string   `mapstructure:"password"`
	IndexChunks           string   `mapstructure:"index_chunks"`
	IndexTickets          string   `mapstructure:"index_tickets"`
	RequestTimeoutSeconds int      `mapstructure:"request_timeout_seconds"`
}

// MinIOConfig 对象存储配置（知识库原文 + 多模态文件）。
type MinIOConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	AccessKey   string `mapstructure:"access_key"`
	SecretKey   string `mapstructure:"secret_key"`
	UseSSL      bool   `mapstructure:"use_ssl"`
	Region      string `mapstructure:"region"`
	BucketKB    string `mapstructure:"bucket_kb"`
	BucketMedia string `mapstructure:"bucket_media"`
}

// RocketMQConfig 消息队列配置（走 broker 的 gRPC proxy）。
type RocketMQConfig struct {
	Enable        bool   `mapstructure:"enable"`
	Endpoint      string `mapstructure:"endpoint"`
	AccessKey     string `mapstructure:"access_key"`
	SecretKey     string `mapstructure:"secret_key"`
	Namespace     string `mapstructure:"namespace"`
	TopicEvents   string `mapstructure:"topic_events"`
	ProducerGroup string `mapstructure:"producer_group"`
	ConsumerGroup string `mapstructure:"consumer_group"`
}

// PrecheckConfig 预检流程配置。
type PrecheckConfig struct {
	MaxSteps          int     `mapstructure:"max_steps"`
	TopK              int     `mapstructure:"top_k"`
	MinScore          float64 `mapstructure:"min_score"`
	SessionTTLMinutes int     `mapstructure:"session_ttl_minutes"`
}

// SLAConfig 时效与升级配置。
type SLAConfig struct {
	ScanIntervalSeconds int            `mapstructure:"scan_interval_seconds"`
	ReminderLeadMinutes int            `mapstructure:"reminder_lead_minutes"`
	Minutes             map[string]int `mapstructure:"minutes"`
}

// Load 读取配置：搜配置文件 + 环境变量覆盖 + 本地兜底 + 校验。
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetEnvPrefix("BOKEONCALL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	setDefaults(v)

	loadedFrom, err := bindConfigFile(v, path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}
	cfg.LoadedFrom = loadedFrom
	cfg.fillLocalDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// bindConfigFile 依次尝试候选配置文件，返回实际加载的文件路径（空表示用内置默认值）。
func bindConfigFile(v *viper.Viper, path string) (string, error) {
	// 显式指定路径：找不到就直接报错，避免"以为读到了、其实没读到"
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("指定的配置文件不可用 %s: %w", path, err)
		}
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return "", fmt.Errorf("读取配置文件失败 %s: %w", path, err)
		}
		return path, nil
	}

	candidates := make([]string, 0, 5)
	if envPath := strings.TrimSpace(os.Getenv("BOKEONCALL_CONFIG")); envPath != "" {
		candidates = append(candidates, envPath)
	}
	candidates = append(candidates,
		filepath.Join("configs", "config.local.yaml"),
		filepath.Join("configs", "config.yaml"),
		"config.local.yaml",
		"config.yaml",
	)

	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		v.SetConfigFile(candidate)
		if err := v.ReadInConfig(); err != nil {
			return "", fmt.Errorf("读取配置文件失败 %s: %w", candidate, err)
		}
		return candidate, nil
	}
	// 一个都没找到：留给 Validate 判断当前环境是否允许（本地放行，非本地报错）
	return "", nil
}

// fillLocalDefaults 本地兜底：模板里留空的凭据用内置默认值补上，保证刚 clone 下来能跑通。
// 非 local 环境不做兜底，Validate 会要求显式配置。
func (c *Config) fillLocalDefaults() {
	if c.App.Env != "local" {
		return
	}
	if c.MySQL.Password == "" {
		c.MySQL.Password = localMySQLPassword
	}
	if c.MinIO.AccessKey == "" {
		c.MinIO.AccessKey = localMinIOAccessKey
	}
	if c.MinIO.SecretKey == "" {
		c.MinIO.SecretKey = localMinIOSecretKey
	}
}

// Validate 启动期校验。
//
// 本地（app.env=local）：只要基本项合理即可，允许留空走内置默认值。
// 非本地：必须显式提供关键凭据（配置文件或环境变量都行，不强制落文件），
// 宁可直接启动失败，也不要带着开发默认密码跑起来。
func (c *Config) Validate() error {
	if c.App.Port <= 0 || c.App.Port > 65535 {
		return fmt.Errorf("app.port 非法: %d", c.App.Port)
	}
	if c.MySQL.Host == "" || c.MySQL.Database == "" {
		return errors.New("mysql.host / mysql.database 不能为空")
	}
	if c.Lark.EventMode != EventModeCallback && c.Lark.EventMode != EventModeLongConn {
		return fmt.Errorf("lark.event_mode 只能是 %s 或 %s，当前值: %q",
			EventModeCallback, EventModeLongConn, c.Lark.EventMode)
	}
	if len(c.SLA.Minutes) == 0 {
		return errors.New("sla.minutes 未配置")
	}
	if c.App.Env == "" || c.App.Env == "local" {
		return nil
	}

	missing := make([]string, 0, 6)
	if !c.Lark.Enabled() {
		missing = append(missing, "lark.app_id / lark.app_secret")
	}
	if c.Lark.VerificationToken == "" {
		missing = append(missing, "lark.verification_token")
	}
	if c.MySQL.Password == "" {
		missing = append(missing, "mysql.password")
	}
	if c.MinIO.AccessKey == "" || c.MinIO.SecretKey == "" {
		missing = append(missing, "minio.access_key / minio.secret_key")
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s 环境缺少必填配置: %s", c.App.Env, strings.Join(missing, "、"))
	}
	return nil
}

// IsProd 是否生产环境。
func (c *Config) IsProd() bool { return c.App.Env == "prod" }

// SLAMinutes 取某优先级的目标时长（分钟）。
func (c *Config) SLAMinutes(priority string) int {
	if v, ok := c.SLA.Minutes[strings.ToLower(priority)]; ok && v > 0 {
		return v
	}
	return 120
}

// setDefaults 内置默认值：只放"通用、不含凭据"的部分。
// 凭据（数据库密码、对象存储 key、飞书 secret）一律不设默认值，本地由 fillLocalDefaults 兜底，
// 非本地环境必须显式提供，避免生产环境静默使用默认密码。
//
// 注意（viper 的坑）：AutomaticEnv + Unmarshal 只对"已知的键"生效——
// 必须先用 SetDefault 把键登记下来（哪怕是空串），环境变量才会被读进来。
// 所以下面给所有"只能靠环境变量提供"的字段设了空默认值，别删。
func setDefaults(v *viper.Viper) {
	v.SetDefault("app.env", "local")
	v.SetDefault("app.host", "0.0.0.0")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.group_prefix", "OnCall")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.dir", "logs")
	v.SetDefault("log.console", true)

	v.SetDefault("mysql.host", "127.0.0.1")
	v.SetDefault("mysql.port", 3307)
	v.SetDefault("mysql.user", "bokeoncall")
	v.SetDefault("mysql.password", "")
	v.SetDefault("mysql.database", "bokeoncall")
	v.SetDefault("mysql.charset", "utf8mb4")
	v.SetDefault("mysql.max_open_conns", 50)
	v.SetDefault("mysql.max_idle_conns", 10)
	v.SetDefault("mysql.auto_migrate", true)

	v.SetDefault("lark.app_id", "")
	v.SetDefault("lark.app_secret", "")
	v.SetDefault("lark.verification_token", "")
	v.SetDefault("lark.encrypt_key", "")
	v.SetDefault("lark.bot_open_id", "")
	// 默认长连接：新同学 clone 下来不用公网入口就能收事件；生产改成 callback
	v.SetDefault("lark.event_mode", EventModeLongConn)
	v.SetDefault("lark.api_base", "https://open.feishu.cn")
	v.SetDefault("lark.timeout_seconds", 10)

	v.SetDefault("es.addresses", []string{"http://127.0.0.1:9200"})
	v.SetDefault("es.username", "")
	v.SetDefault("es.password", "")
	v.SetDefault("es.index_chunks", "bokeoncall_kb_chunks")
	v.SetDefault("es.index_tickets", "bokeoncall_tickets")
	v.SetDefault("es.request_timeout_seconds", 10)

	v.SetDefault("minio.endpoint", "127.0.0.1:9000")
	v.SetDefault("minio.access_key", "")
	v.SetDefault("minio.secret_key", "")
	v.SetDefault("minio.use_ssl", false)
	v.SetDefault("minio.region", "us-east-1")
	v.SetDefault("minio.bucket_kb", "bokeoncall")
	v.SetDefault("minio.bucket_media", "bokeoncall-media")

	v.SetDefault("rocketmq.enable", false)
	v.SetDefault("rocketmq.endpoint", "127.0.0.1:8081")
	v.SetDefault("rocketmq.access_key", "")
	v.SetDefault("rocketmq.secret_key", "")
	v.SetDefault("rocketmq.namespace", "")
	v.SetDefault("rocketmq.topic_events", "bokeoncall_events")
	v.SetDefault("rocketmq.producer_group", "bokeoncall-producer")
	v.SetDefault("rocketmq.consumer_group", "bokeoncall-consumer")

	v.SetDefault("precheck.max_steps", 5)
	v.SetDefault("precheck.top_k", 5)
	v.SetDefault("precheck.min_score", 0.35)
	v.SetDefault("precheck.session_ttl_minutes", 120)

	v.SetDefault("sla.scan_interval_seconds", 60)
	v.SetDefault("sla.reminder_lead_minutes", 10)
	v.SetDefault("sla.minutes", map[string]int{"p0": 15, "p1": 30, "p2": 120, "p3": 480})
}

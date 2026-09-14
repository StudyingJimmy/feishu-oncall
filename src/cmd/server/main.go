// Command server HTTP 服务入口：接收飞书事件（回调或长连接）并交给路由处理。
//
// 启动流程：加载配置 → 初始化日志 → 按配置启动事件接入 → 起 HTTP 服务（健康检查 / 回调地址）→ 优雅退出。
// 依赖装配（MySQL / ES / 对象存储等）在 internal/app，路由在 internal/api 与 internal/service/router。
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"bokeoncall/internal/api"
	"bokeoncall/internal/conf"
	"bokeoncall/internal/infra/lark"
	"bokeoncall/internal/service"
	"bokeoncall/internal/service/precheck"
	"bokeoncall/internal/service/router"
	"bokeoncall/pkg/logx"
)

func main() {
	configPath := flag.String("config", "", "配置文件路径，默认依次找 configs/config.yaml")
	flag.Parse()

	cfg, err := conf.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "加载配置失败:", err)
		os.Exit(1)
	}
	if err := logx.Init(logx.Options{Level: cfg.Log.Level, Dir: cfg.Log.Dir, Console: cfg.Log.Console}); err != nil {
		fmt.Fprintln(os.Stderr, "初始化日志失败:", err)
		os.Exit(1)
	}
	defer logx.Sync()
	logConfigSource(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 飞书客户端：主动调接口用（后续 handle 实现会用到），同时用来拿机器人 open_id
	larkClient := lark.NewClient(cfg.Lark)
	botOpenID := resolveBotOpenID(ctx, larkClient)

	// TODO(依赖装配): 在这里初始化 MySQL / ES / 对象存储等，并注入到 service.NewHandler。
	precheckSvc := precheck.NewService(cfg, larkClient)
	eventRouter := router.New(botOpenID, service.NewHandler(precheckSvc))
	callbacks := startEventSource(ctx, cfg, eventRouter)

	srv := &http.Server{
		Addr:              cfg.App.Address(),
		Handler:           api.New(callbacks),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		logx.L().Info("HTTP 服务启动", zap.String("addr", cfg.App.Address()), zap.String("env", cfg.App.Env))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logx.L().Error("HTTP 服务异常退出", zap.Error(err))
			stop()
		}
	}()

	<-ctx.Done()
	logx.L().Info("收到退出信号，开始优雅关闭")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logx.L().Error("优雅关闭失败", zap.Error(err))
	}
}

// startEventSource 按配置启动事件接入，返回回调模式下要挂到 HTTP 路由上的处理器。
func startEventSource(ctx context.Context, cfg *conf.Config, eventRouter *router.Router) http.Handler {
	if !cfg.Lark.Enabled() {
		logx.L().Warn("未配置飞书应用凭证（lark.app_id / app_secret），事件接入不会真正生效")
	}
	if cfg.Lark.IsLongConn() {
		if cfg.IsProd() {
			logx.L().Warn("生产环境建议用 callback 模式，长连接主要用于本地调试")
		}
		client := lark.NewLongConnClient(cfg.Lark, eventRouter.Handle)
		go func() {
			if err := client.Start(ctx); err != nil && ctx.Err() == nil {
				logx.L().Error("飞书长连接退出", zap.Error(err))
			}
		}()
		return nil
	}

	logx.L().Info("事件接入方式：HTTP 回调", zap.String("path", "/lark/event"))
	return lark.NewCallbackHandler(cfg.Lark, eventRouter.Handle)
}

// resolveBotOpenID 取机器人 open_id：配置里写了直接用，没写就用 app_id / app_secret 调飞书获取。
// 拿不到时返回空串，路由会退化为「@了任何人就算 @机器人」。
func resolveBotOpenID(ctx context.Context, client *lark.Client) string {
	if !client.Enabled() {
		logx.L().Warn("未配置 lark.app_id / app_secret，跳过机器人 open_id 获取")
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	openID, err := client.EnsureBotOpenID(ctx)
	if err != nil {
		logx.L().Warn("自动获取机器人 open_id 失败，群里「是否 @机器人」会退化为「是否 @了任何人」", zap.Error(err))
		return ""
	}
	logx.L().Info("机器人 open_id 已就绪", zap.String("bot_open_id", openID))
	return openID
}

// logConfigSource 提示配置来源，避免"以为读到了配置，其实用的内置默认值"。
func logConfigSource(cfg *conf.Config) {
	switch {
	case cfg.LoadedFrom != "":
		logx.L().Info("配置加载完成",
			zap.String("config", cfg.LoadedFrom),
			zap.String("env", cfg.App.Env),
			zap.String("event_mode", cfg.Lark.EventMode),
		)
	case cfg.App.Env == "local":
		logx.L().Warn("未找到配置文件，使用内置默认值 + 环境变量（本地够用）；建议执行 make init-config 生成 configs/config.yaml")
	default:
		logx.L().Info("未使用配置文件，配置全部来自内置默认值与环境变量", zap.String("env", cfg.App.Env))
	}
}

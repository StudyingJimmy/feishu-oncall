// Command worker 后台进程入口：定时任务 + 消息队列消费。
//
// 当前只做启动该做的事：加载配置 -> 初始化日志 -> 等退出信号。
// 定时任务放 internal/task（待实现），消息队列客户端已封装在 internal/infra/mq。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"bokeoncall/internal/conf"
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

	switch {
	case cfg.LoadedFrom != "":
		logx.L().Info("配置加载完成", zap.String("config", cfg.LoadedFrom), zap.String("env", cfg.App.Env))
	case cfg.App.Env == "local":
		logx.L().Warn("未找到配置文件，使用内置默认值 + 环境变量（本地够用）；建议执行 make init-config 生成 configs/config.yaml")
	default:
		logx.L().Info("未使用配置文件，配置全部来自内置默认值与环境变量", zap.String("env", cfg.App.Env))
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// TODO(任务装配): 在这里启动后台任务，例如
	//   go task.NewSLAWatchdog(...).Run(ctx)   // 超时工单扫描与自动升级
	//   go task.NewRosterSync(...).Run(ctx)    // 值班表同步
	//   consumer := mq.NewConsumer(cfg.RocketMQ, handler)  // handler 在 internal/task 里实现
	//   go consumer.Start(ctx)

	logx.L().Info("worker 已启动（定时任务与事件消费待实现）")
	<-ctx.Done()
	logx.L().Info("收到退出信号，worker 关闭中")
}

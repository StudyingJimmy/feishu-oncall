// Package logx 统一日志：控制台 + 按大小滚动的文件。
//
// 约定：各层通过 logx.L() 取全局 logger，不要在业务代码里 fmt.Println 打日志。
package logx

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	mu     sync.RWMutex
	logger *zap.Logger
)

// Options 初始化参数，由 conf.LogConfig 转换而来。
type Options struct {
	Level   string // debug / info / warn / error
	Dir     string // 日志目录，为空则只输出到控制台
	Console bool
}

// Init 初始化全局 logger，服务启动时调用一次。
func Init(opt Options) error {
	level, err := zapcore.ParseLevel(opt.Level)
	if err != nil {
		level = zapcore.InfoLevel
	}

	encCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stack",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	cores := make([]zapcore.Core, 0, 2)
	if opt.Console {
		cores = append(cores, zapcore.NewCore(
			zapcore.NewConsoleEncoder(encCfg),
			zapcore.AddSync(os.Stdout),
			level,
		))
	}
	if opt.Dir != "" {
		if err := os.MkdirAll(opt.Dir, 0o755); err != nil {
			return fmt.Errorf("创建日志目录失败: %w", err)
		}
		writer := &lumberjack.Logger{
			Filename:   filepath.Join(opt.Dir, "bokeoncall.log"),
			MaxSize:    100, // MB
			MaxBackups: 10,
			MaxAge:     30, // 天
			Compress:   true,
		}
		cores = append(cores, zapcore.NewCore(
			zapcore.NewJSONEncoder(encCfg),
			zapcore.AddSync(writer),
			level,
		))
	}
	if len(cores) == 0 {
		cores = append(cores, zapcore.NewCore(zapcore.NewConsoleEncoder(encCfg), zapcore.AddSync(os.Stdout), level))
	}

	l := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	mu.Lock()
	logger = l
	mu.Unlock()
	return nil
}

// L 返回全局 logger；未初始化时返回一个只输出到控制台的实例。
func L() *zap.Logger {
	mu.RLock()
	l := logger
	mu.RUnlock()
	if l != nil {
		return l
	}
	fallback, _ := zap.NewProduction()
	return fallback
}

// Sync 刷盘，进程退出前调用。
func Sync() {
	_ = L().Sync()
}

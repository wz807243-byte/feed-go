package logger

import (
	"go.uber.org/zap"
)

var logger *zap.SugaredLogger

func Init() {
	l, err := zap.NewDevelopmentConfig().Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(l)
	logger = l.WithOptions(zap.AddCallerSkip(1)).Sugar()
}

func get() *zap.SugaredLogger {
	if logger == nil {
		Init()
	}
	return logger
}

func Println(args ...any) {
	get().Info(args...)
}

func Printf(template string, args ...any) {
	get().Infof(template, args...)
}

// Fatalf 与标准库 log.Fatalf 行为一致：输出日志后以退出码 1 终止进程
func Fatalf(template string, args ...any) {
	get().Fatalf(template, args...)
}

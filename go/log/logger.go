package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// 全局变量，方便直接调用
var (
	Debug func(args ...interface{})
	Info  func(args ...interface{})
	Warn  func(args ...interface{})
	Error func(args ...interface{})
	Fatal func(args ...interface{})

	Debugf func(template string, args ...interface{})
	Infof  func(template string, args ...interface{})
	Warnf  func(template string, args ...interface{})
	Errorf func(template string, args ...interface{})
)

var once sync.Once

// Init 初始化日志系统
func Init() {
	once.Do(func() {
		// 创建日志目录
		logDir := "logs"
		if err := os.MkdirAll(logDir, 0755); err != nil {
			panic(err)
		}

		// 日志文件
		logFile, _ := os.OpenFile(
			filepath.Join(logDir, time.Now().Format("2006-01-02")+".log"),
			os.O_CREATE|os.O_APPEND|os.O_WRONLY,
			0644,
		)

		encoderConfig := zapcore.EncoderConfig{
			TimeKey:       "T",
			LevelKey:      "L",
			NameKey:       "N",
			CallerKey:     "C",
			FunctionKey:   zapcore.OmitKey,
			MessageKey:    "M",
			StacktraceKey: "S",
			LineEnding:    zapcore.DefaultLineEnding,
			EncodeLevel:   zapcore.CapitalLevelEncoder,
			EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
				enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
			},
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig), // 使用Console编码器替代JSON编码器
			zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(logFile)),
			zap.NewAtomicLevelAt(zap.DebugLevel),
		)

		logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)).Sugar()

		Debug = logger.Debug
		Info = logger.Info
		Warn = logger.Warn
		Error = logger.Error

		Debugf = logger.Debugf
		Infof = logger.Infof
		Warnf = logger.Warnf
		Errorf = logger.Errorf
	})
}

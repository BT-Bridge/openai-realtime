package shared

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type LoggerAdapter interface {
	Error(msg string, err error, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
	Trace(msg string, fields ...zap.Field)
	With(fields ...zap.Field) LoggerAdapter
}

type stdLogger struct {
	logger *zap.Logger
}

var _ LoggerAdapter = (*stdLogger)(nil)

func (s *stdLogger) Error(msg string, err error, fields ...zap.Field) {
	s.logger.Error(msg, append(fields, zap.Error(err))...)
}

func (s *stdLogger) Info(msg string, fields ...zap.Field) {
	s.logger.Info(msg, fields...)
}

func (s *stdLogger) Debug(msg string, fields ...zap.Field) {
	s.logger.Debug(msg, fields...)
}

func (s *stdLogger) Trace(msg string, fields ...zap.Field) {
	s.logger.Debug(msg, fields...)
}

func (s *stdLogger) With(fields ...zap.Field) LoggerAdapter {
	return &stdLogger{logger: s.logger.With(fields...)}
}

func NewStdLogger() LoggerAdapter {
	logger, err := zap.NewProduction(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
	return &stdLogger{logger: logger}
}

type fileLogger struct {
	logger *zap.Logger
}

var _ LoggerAdapter = (*fileLogger)(nil)

func (f *fileLogger) Error(msg string, err error, fields ...zap.Field) {
	f.logger.Error(msg, append(fields, zap.Error(err))...)
}

func (f *fileLogger) Info(msg string, fields ...zap.Field) {
	f.logger.Info(msg, fields...)
}

func (f *fileLogger) Debug(msg string, fields ...zap.Field) {
	f.logger.Debug(msg, fields...)
}

func (f *fileLogger) Trace(msg string, fields ...zap.Field) {
	f.logger.Debug(msg, fields...)
}

func (f *fileLogger) With(fields ...zap.Field) LoggerAdapter {
	return &fileLogger{logger: f.logger.With(fields...)}
}

func NewFileLogger(filename string, maxSizeMB int, maxBackups int, maxAgeDays int, compress bool) LoggerAdapter {
	hook := lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
		MaxAge:     maxAgeDays,
		Compress:   compress,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&hook),
		zapcore.DebugLevel,
	)

	logger := zap.New(core, zap.AddCallerSkip(1))
	return &fileLogger{logger: logger}
}

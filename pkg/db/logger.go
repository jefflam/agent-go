package db

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"

	"github.com/lisanmuaddib/agent-go/pkg/logging"
)

// GormLogrusLogger implements GORM's logger.Interface using logrus
type GormLogrusLogger struct {
	logger        *logrus.Logger
	slowThreshold time.Duration
}

// NewGormLogrusLogger creates a new GORM logger that uses logrus
func NewGormLogrusLogger(baseLogger *logrus.Logger) *GormLogrusLogger {
	if _, ok := baseLogger.Formatter.(*logging.ColoredJSONFormatter); !ok {
		baseLogger.SetFormatter(logging.NewColoredJSONFormatter())
	}

	return &GormLogrusLogger{
		logger:        baseLogger,
		slowThreshold: 200 * time.Millisecond,
	}
}

// LogMode implements logger.Interface
func (l *GormLogrusLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

// Info implements logger.Interface
func (l *GormLogrusLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	l.logger.WithContext(ctx).WithFields(logrus.Fields{
		"source": "gorm",
		"type":   "query_info",
	}).Debugf(msg, args...)
}

// Warn implements logger.Interface
func (l *GormLogrusLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	l.logger.WithContext(ctx).WithFields(logrus.Fields{
		"source": "gorm",
		"type":   "query_warn",
	}).Warnf(msg, args...)
}

// Error implements logger.Interface
func (l *GormLogrusLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	l.logger.WithContext(ctx).WithFields(logrus.Fields{
		"source": "gorm",
		"type":   "query_error",
	}).Errorf(msg, args...)
}

// Trace implements logger.Interface
func (l *GormLogrusLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := logrus.Fields{
		"source":   "gorm",
		"type":     "query_trace",
		"elapsed":  elapsed,
		"rows":     rows,
		"sql":      sql,
		"duration": elapsed.String(),
	}

	if err != nil {
		fields["error"] = err
		l.logger.WithContext(ctx).WithFields(fields).Error("database query failed")
		return
	}

	if elapsed > l.slowThreshold {
		l.logger.WithContext(ctx).WithFields(fields).Warn("slow query detected")
		return
	}

	l.logger.WithContext(ctx).WithFields(fields).Debug("database query executed")
}

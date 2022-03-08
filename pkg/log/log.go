package log

import (
	"fmt"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"

	"github.com/projecteru2/yavirt/internal/errors"
)

// Setup .
func Setup(level, file, sentryDSN string) (func(), error) {
	if err := setupLevel(level); err != nil {
		return nil, errors.Trace(err)
	}

	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	if err := setupOutput(file); err != nil {
		return nil, errors.Trace(err)
	}

	return setupSentry(sentryDSN)
}

func setupSentry(dsn string) (func(), error) {
	if len(dsn) == 0 {
		return func() {}, nil
	}

	deferSentry := func() {
		defer sentry.Flush(time.Second * 2) //nolint
		if err := recover(); err != nil {
			sentry.CaptureMessage(fmt.Sprintf("%v", err))
			panic(err)
		}
	}

	return deferSentry, sentry.Init(sentry.ClientOptions{Dsn: dsn})
}

func setupOutput(file string) error {
	if len(file) < 1 {
		return nil
	}

	var f, err = os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return errors.Trace(err)
	}

	logrus.SetOutput(f)

	return nil
}

func setupLevel(level string) error {
	if len(level) < 1 {
		return nil
	}

	var lv, err = logrus.ParseLevel(level)
	if err != nil {
		return errors.Trace(err)
	}

	logrus.SetLevel(lv)

	return nil
}

// WarnStackf .
func WarnStackf(err error, format string, args ...interface{}) {
	WarnStack(errors.Annotatef(err, format, args...))
}

// WarnStack .
func WarnStack(err error) {
	Warnf(errors.Stack(err))
}

// Warnf .
func Warnf(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

// ErrorStackf .
func ErrorStackf(err error, format string, args ...interface{}) {
	ErrorStack(errors.Annotatef(err, format, args...))
}

// ErrorStack .
func ErrorStack(err error) {
	Errorf(errors.Stack(err))
}

// Errorf .
func Errorf(format string, args ...interface{}) {
	sentry.CaptureMessage(fmt.Sprintf(format, args...))
	logrus.Errorf(format, args...)
}

// Infof .
func Infof(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

// Debugf .
func Debugf(format string, args ...interface{}) {
	logrus.Debugf(format, args...)
}

// FatalStack .
func FatalStack(err error) {
	Fatalf(errors.Stack(err))
}

// Fatalf .
func Fatalf(format string, args ...interface{}) {
	sentry.CaptureMessage(fmt.Sprintf(format, args...))
	logrus.Fatalf(format, args...)
}

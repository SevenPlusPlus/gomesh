package log

import (
	"testing"
	"time"
)

func TestDefaultLogger(t *testing.T) {
	DefaultLogger.Debugf("hello test logger")
	DefaultLogger.Infof("Hello, current time is %v\n", time.Now())
}

func TestFileLogger(t *testing.T) {
	InitDefaultLogger("../../logs/mylog.log", DEBUG)
	DefaultLogger.Debugf("hello test logger")
	DefaultLogger.Infof("Hello, current time is %v\n", time.Now())
}
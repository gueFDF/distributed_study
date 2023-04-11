package lg

import (
	"testing"
)

func TestLog(t *testing.T) {
	//SetFlags(log.Ldate | log.Lmicroseconds | log.Llongfile)
	Config(INFO, nil)
	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")
	Fatal("fatal")
}

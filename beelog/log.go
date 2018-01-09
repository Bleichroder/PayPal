package beelog

import "github.com/astaxie/beego/logs"

var Log *logs.BeeLogger

func InitLog() {
	Log = logs.NewLogger(1000)
	Log.EnableFuncCallDepth(true)
}

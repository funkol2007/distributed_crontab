package main

import (
	"fmt"
	"helper/common"
	logger "github.com/shengkehua/xlog4go"
	"runtime/debug"
)

//退出信号
var G_QuitChan = make(chan int)

func main() {
	//初始化线程
	initEvn()

	//初始化log
	if err := initLog(); err != nil {
		fmt.Errorf("init_log_fail errno:%d errmsg:%s\n", common.INIT_LOG_FAILED, err.Error())
		return
	}
	defer logger.Close()

	//设置recover
	defer func() {
		if err := recover(); err != nil {
			logger.Error("abort, unknown error, errno:%d,errmsg:%v, stack:%s",
				common.ERRNO_PANIC, err, string(debug.Stack()))
		}
	}()

	//初始化config
	if err := initConf(); err != nil {
		logger.Warn("init_conf_fail errno:%d errmsg:%s\n", common.INIT_SERVCIE_FAILED, err.Error())
		return
	}

	//初始化jobmgr
	if err := initJobMgr();err !=nil {
		logger.Warn("init_jobmgr errno:%d errmsg:%s\n",common.INIT_SERVCIE_FAILED,err.Error())
		return
	}

	//初始化日志收集器
	if err := initLogMgr();err != nil {
		logger.Warn("init_logmgr errno:%d errmsg:%s\n",common.INIT_SERVCIE_FAILED,err.Error())
		return
	}
	//初始化调度器
	if err := initScheduler();err != nil {
		logger.Warn("init_scheduler errno:%d errmsg=%s\n", common.INIT_SERVCIE_FAILED, err.Error())
		return
	}

	//初始化执行器
	if err := initExecutor();err != nil {
		logger.Warn("init_executor errno:%d errmsg=%s\n", common.INIT_SERVCIE_FAILED, err.Error())
		return
	}
	//初始化watcher,启动监听
	if err := initWatcher();err != nil {
		logger.Warn("init_conf_watcher errno:%d errmsg:%s\n",common.INIT_SERVCIE_FAILED,err.Error())
		return
	}

	if err := initRegiter();err != nil {
		logger.Warn("init_register errno:%d errmsg:%s\n",common.INIT_SERVCIE_FAILED,err.Error())
		return
	}

	logger.Info("all_init_ok")
	fmt.Println("start_ok")

	//监听中断信号
	go signal_proc()

	value := <-G_QuitChan

	logger.Info("msg:diversion_api_quit chan_recv_val:%d", value)
	return

}
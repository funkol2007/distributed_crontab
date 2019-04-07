package main

import (
	"runtime"
	"worker/config"
	"fmt"
	"helper/logid"
	"helper/common"
	logger "github.com/shengkehua/xlog4go"
	"github.com/BurntSushi/toml"
	"worker/jobmgr"
	"worker/logmgr"
	"worker/register"
)

func initEvn() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func initLog() error {
	if err := logger.SetupLogWithConf(config.LogFile); err != nil {
		fmt.Println("log init fail: %s", err.Error())
		return err
	}

	logid.LogId = common.NowInNs()

	logger.Info("init logger success.")
	return nil
}

func initConf() error {
	_, err := toml.DecodeFile(config.ConfFile, &config.Cfg)
	if err != nil {
		fmt.Println("failed to parse conf:%s", err.Error())
		return err
	}
	logger.Info("config: %v", config.Cfg)
	logger.Info("init cfg success.")
	return nil
}

func initJobMgr() error {
	return jobmgr.Init()
}

func initWatcher() error {
	e1 := jobmgr.G_JobMgr.WatchJobs()
	e2 := jobmgr.G_JobMgr.WatchKillJobs()
	if e1 != nil {
		return e1
	}else {
		return e2
	}
}

func initScheduler() error {
	return jobmgr.InitScheduler()
}

func initExecutor() error {
	return jobmgr.InitExecutor()
}

func initLogMgr() error {
	return logmgr.InitLogMgr()
}

func initRegiter() error {
	return register.InitRegiter()
}
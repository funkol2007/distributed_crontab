package main

import (
	"helper/common"
	"fmt"
	"helper/logid"
	logger "github.com/shengkehua/xlog4go"
	"server/httpserver"
	"runtime"
	"master/config"
	"github.com/BurntSushi/toml"
	"master/jobmgr"
	"master/logmgr"
	"master/workermgr"
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

func initHttpServer() error {
	logger.Info("init http")
	HttpInstance := httpserver.GetHttpInstance()
	if err := HttpInstance.Init(config.Cfg.Http.Port); err != nil {
		logger.Warn("init_http_server_failed")
		return err
	}

	for uri, handler := range httpserver.Uri2Handler {
		HttpInstance.AddHandler(uri, handler)
	}

	if err := HttpInstance.Start(); err != nil {
		logger.Fatal("start_http_server_failed")
		return err
	}

	logger.Info("init httpserver success.")
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

func initLogMgr() error {
	return logmgr.InitLogMgr()
}

func initWorkerMgr() error {
	return workermgr.InitWorkerMgr()
}
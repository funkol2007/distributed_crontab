package logmgr

import (
	"go.mongodb.org/mongo-driver/mongo"
	"time"
	"go.mongodb.org/mongo-driver/mongo/options"
	"context"
	"worker/config"
	"helper/common"
	logger "github.com/shengkehua/xlog4go"
)

//log管理器
type LogMgr struct {
	Client *mongo.Client //连接mongodb的客户端
	LogChan chan *common.CronLog //写入日志的chan
	Collection *mongo.Collection //mongodb的表
}

var (
	G_LogMgr  *LogMgr
)

func InitLogMgr() (err error) {
	//连接mogodb
	var (
		client *mongo.Client
	)
	con := context.TODO()
	//1、建立连接
	opt := options.Client()
	opt.SetConnectTimeout(time.Duration(config.Cfg.LogMgr.TimeOut)*time.Millisecond).ApplyURI(config.Cfg.LogMgr.MongodbUri)
	if client,err = mongo.Connect(con,opt); err != nil {
		return
	}

	G_LogMgr = &LogMgr{
		Client:client,
		LogChan:make(chan *common.CronLog,common.MAX_NUM_LOG_QUEUE),
		Collection:client.Database(config.Cfg.LogMgr.Database).Collection(config.Cfg.LogMgr.Collection),
	}

	//启动监听器
	go G_LogMgr.LogLoop()
	return
}

//监听循环
func (l *LogMgr) LogLoop() {

	//初始化timer
	var (
		commitTimer = time.NewTimer(time.Duration(config.Cfg.LogMgr.LogTimeOut)*time.Second)
		logBatch = &common.LogBach{}
	)
	for {
		select {
			case v := <-l.LogChan:
				logBatch.Logs = append(logBatch.Logs,v)
				if len(logBatch.Logs) >= config.Cfg.LogMgr.MaxBatchSize { //达到batchsize 执行一次存储
					//满了以后先暂停计时器
					commitTimer.Reset(time.Duration(config.Cfg.LogMgr.LogTimeOut)*time.Second)
					commitTimer.Stop()
					l.DoLogSave(logBatch)
					logBatch.Logs = make([]interface{},0)
				}
				case <-commitTimer.C:
					commitTimer.Reset(time.Duration(config.Cfg.LogMgr.LogTimeOut)*time.Second)
					if len(logBatch.Logs) != 0 {
						l.DoLogSave(logBatch)
						logBatch.Logs = make([]interface{},0)
					}
		}
	}
}

func (l *LogMgr) DoLogSave(batch *common.LogBach) {
	//插入多个
	if _, err := l.Collection.InsertMany(context.TODO(),batch.Logs);err != nil {
		logger.Error("Log Save Error errno:%d, err:%s", common.ERRNO_LOG_SAVE_FAILED, err.Error())
	}

	return
}
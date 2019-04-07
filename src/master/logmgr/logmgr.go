package logmgr

import (
"go.mongodb.org/mongo-driver/mongo"
"time"
"go.mongodb.org/mongo-driver/mongo/options"
"context"
"master/config"
"helper/common"
)

//log管理器
type LogMgr struct {
	Client *mongo.Client //连接mongodb的客户端
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
		Collection:client.Database(config.Cfg.LogMgr.Database).Collection(config.Cfg.LogMgr.Collection),
	}

	return
}

func (l *LogMgr) ListLog(name string,skip int,limit int) (logArr []*common.CronLog, err error){
	var(
		cursor *mongo.Cursor
		con context.Context
	)

	//初始化返回
	logArr = make([]*common.CronLog,0)

	filter := &common.LogFilter{Name:name}
	logsort := &common.SortLogByStartTime{SortOrder:-1}
	skip64 := int64(skip)
	limit64 := int64(limit)
	findOpt := &options.FindOptions{
		Limit:&limit64,
		Skip:&skip64,
		Sort:logsort,
	}
	if cursor, err = l.Collection.Find(con,filter,findOpt);err != nil {
		return
	}
	defer cursor.Close(con)
	for cursor.Next(con) {
		jobLog := &common.CronLog{}

		//反序列化bson
		if err = cursor.Decode(jobLog);err != nil {
			//日志不合法跳过本条
			continue
		}
		logArr = append(logArr,jobLog)
	}
	return
}


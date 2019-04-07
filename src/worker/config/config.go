package config

var (
	LogFile     = "./src/worker/config/log.json"
	ConfFile    = "./src/worker/config/worker.conf"
)

type WorkerCfg struct {
	JobMgr JobMgrConfig
	LogMgr LogMgrConfig
}

type JobMgrConfig struct{
	Endpoints []string
	TimeOut int
}


type LogMgrConfig struct{
	MongodbUri string
	TimeOut int
	Database string
	Collection string
	MaxBatchSize int
	LogTimeOut int
}

var Cfg WorkerCfg
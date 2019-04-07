package config

var (
	LogFile     = "./src/master/config/log.json"
	ConfFile    = "./src/master/config/master.conf"
	WebrootPath = "./src/master/webroot"
)

type MasterCfg struct {
	Http   HttpConfig
	JobMgr JobMgrConfig
	LogMgr LogMgrConfig
}

type HttpConfig struct {
	Port string
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
var Cfg MasterCfg

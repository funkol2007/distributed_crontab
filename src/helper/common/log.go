package common


type CronLog struct{
	Name string `bson:"name" json:"name"`//任务名
	Command string `bson:"command" json:"command"`//命令
	OutPut string `bson:"output" json:"output"`//输出结果
	Err string `bson:"err" json:"err"`//错误
	StartTime int64 `bson:"startTime" json:"startTime"` //任务开始时间
	EndTime int64 `bson:"endTime" json:"endTime"` //任务结束时间
	ScheduleTime int64 `bson:"scheduleTime" json:"scheduleTime"` //计划调度时间
}

type LogBach struct {
	Logs []interface{}
}

type LogFilter struct {
	Name string `bson:"name"`
}

type SortLogByStartTime struct {
	SortOrder int `bson:"startTime"` //按照startTime倒序排列
}
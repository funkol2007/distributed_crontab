package jobmgr

type Job struct {
	Name        string `json:"name"`      //任务名
	Command     string `json:"command"`   //shell命令
	CronExpr    string `json:"cronExpr"`  //cron表达式
}

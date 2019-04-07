package common

import (
	"encoding/json"
	"strings"
)

type Job struct {
	Name        string `json:"name"`      //任务名
	Command     string `json:"command"`   //shell命令
	CronExpr    string `json:"cronExpr"`  //cron表达式
}

func UnpackJob(val []byte) (ret *Job, err error) {
	ret = &Job{}
	err = json.Unmarshal(val,ret)
	return
}

func ExtractJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey,JOB_KEY_PREFIX)
}

func ExtractKillJobName(jobKey string) string {
	return strings.TrimPrefix(jobKey,JOB_KILL_PREFIX)
}

type JobEvent struct {
	Type int
	Job *Job
}

func BuildEvent(evenType int, job *Job) *JobEvent {
	return &JobEvent{
		Type:evenType,
		Job:job,
	}
}
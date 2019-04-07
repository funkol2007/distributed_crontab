package jobmgr

import (
	"time"
	"os/exec"
	"helper/common"
	"math/rand"
	logger "github.com/shengkehua/xlog4go"
)

//执行器
var (
	G_Executor *Executor
)

type Executor struct {
	ScheduleResultChan chan *ExecResult
}


//执行一个shell任务
func (e *Executor) ExecJob(plan *SchedulePlan){
	//要在这里赋值上scheduletime计划调度时间。因为plan是指针，后面go并发以后plan的nexttime会改动。
	var scheduleTime = plan.NextTime
	go func(){
		var (
			err error
			result *ExecResult
		)
		//获取乐观锁，成功再往下执行
		jobLock := G_JobMgr.NewJobLock(plan.Job.Name)
		//随机睡眠0-10ms，防止机器时间不同步导致任务分配不均匀
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
		err = jobLock.TryLock()
		defer jobLock.UnLock()
		if err !=nil {
			//上锁失败不能返回，也要更新执行表吧任务从执行表里去掉
			result = &ExecResult{
				Err:err,
				EndTime:time.Now(),
				JobPlan:plan,
			}
		}else {
			//上锁成功
			logger.Info("上锁成功！可以执行任务：%s, command:%s",plan.Job.Name,plan.Job.Command)
			//执行调度
			startTime := time.Now()
			cmd := exec.CommandContext(plan.ctx,"/bin/bash","-c",plan.Job.Command)
			output,err := cmd.CombinedOutput()
			//返回结果
			result = &ExecResult{
				Err:err,
				OutPut:output,
				JobPlan:plan,
				StartTime:startTime,
				EndTime:time.Now(),
				ScheduleTime:scheduleTime,
			}
		}
		e.PushJobResult(result)
	}()
}


func InitExecutor() (err error){
	G_Executor = &Executor{
		ScheduleResultChan :make(chan *ExecResult,common.MAX_NUM_JOB_QUEUE),
	}
	return
}

type ExecResult struct {
	Err error
	OutPut []byte
	JobPlan *SchedulePlan
	StartTime time.Time
	EndTime time.Time
	ScheduleTime time.Time
}

func (e *Executor) PushJobResult(result *ExecResult) {
	e.ScheduleResultChan<-result
}
package jobmgr

import (
	"helper/common"
	"github.com/gorhill/cronexpr"
	logger "github.com/shengkehua/xlog4go"
	"time"
	"fmt"
	"context"
	"worker/logmgr"
)

type Scheduler struct {
	jobEventChan chan *common.JobEvent           //job触发事件，包括删除、更改
	jobEventPlanMap map[string]*SchedulePlan     //job执行计划表
	jobExecMap map[string]*SchedulePlan       //正在执行中的job
}

type SchedulePlan struct {
	Job *common.Job //任务信息
	Expr *cronexpr.Expression  //解析好的cronexpr表达式
	NextTime time.Time //下次执行时间
	ctx context.Context
	cancelFunc context.CancelFunc
}

var (
	G_Scheduler *Scheduler
)

//调度协程
func (sc *Scheduler) scheduleLoop() {
	//定时执行调度commonJob
	var (
		scheduleAfter = sc.trySchedule()
		schedulerTimer = time.NewTimer(scheduleAfter)
	)

	for {
		select {
		case v := <- sc.jobEventChan: //任务更新
			//这里不能并发，因为有map的写入操作
			sc.handle(v)
		case <-schedulerTimer.C: //休眠结束
		case result := <-G_Executor.ScheduleResultChan: //任务执行完毕
			sc.processScheduleResult(result)
		}
		//更新或者睡眠到了（无任务1秒唤醒一次）重新计算时间
		scheduleAfter = sc.trySchedule()
		schedulerTimer.Reset(scheduleAfter)
	}
}

func (sc *Scheduler) processScheduleResult(result *ExecResult) {
	//执行完毕（无论是否成功）需要删除执行列表(这里没有并发了，可以安全删除map的元素)
	delete(sc.jobExecMap,result.JobPlan.Job.Name)
	//任务执行太快可能导致因为机器时间差造成的重复执行。
	time.Sleep(20*time.Millisecond)

	//任务执行成功
	fmt.Println("任务名：",result.JobPlan.Job.Name," 开始时间：",
		result.StartTime," 结束时间：",result.EndTime," 执行结果：",string(result.OutPut))
	//写日志
	LogOne := &common.CronLog{
		Name:result.JobPlan.Job.Name,
		Command:result.JobPlan.Job.Command,
		OutPut:string(result.OutPut),
		StartTime:result.StartTime.UnixNano()/1000/1000,
		EndTime:result.EndTime.UnixNano()/1000/1000,
		ScheduleTime:result.ScheduleTime.UnixNano()/1000/1000,
	}

	if result.Err != nil {
		LogOne.Err = result.Err.Error()
	}else{
		LogOne.Err = ""
	}
	logmgr.G_LogMgr.LogChan <-LogOne
}

//执行并返回要最少休眠的时间
func (sc *Scheduler) trySchedule() (timeAfter time.Duration){
	var (
		now time.Time
		nearestTime *time.Time
	)

	//任务表为空时休眠1s
	if len(sc.jobEventPlanMap) == 0 {
		timeAfter = 1*time.Second
		return
	}
	//初始化时间
	now = time.Now()
	//遍历任务
	for _,v := range sc.jobEventPlanMap{
		//需要执行任务
		if v.NextTime.Before(now)||v.NextTime.Equal(now) {
			//这里也不能并发操作，我这里开始犯了错误，因为内部有map的写操作。
			sc.tryStartJob(v)
			fmt.Println(v.NextTime)
			//执行完以后需要记录下次执行时间
			v.NextTime = v.Expr.Next(now)
			fmt.Println(v.NextTime)
		}
		//统计最近要过期的时间
		if nearestTime == nil || v.NextTime.Before(*nearestTime) {
			nearestTime = &v.NextTime
		}
	}
	//下次调度时间
	timeAfter = nearestTime.Sub(now)
	return
}

//初始化
func InitScheduler() (err error) {
	G_Scheduler = &Scheduler{
		jobEventChan:make(chan *common.JobEvent,common.MAX_NUM_JOB_QUEUE),
		jobEventPlanMap:make(map[string]*SchedulePlan,0),
		jobExecMap: make(map[string]*SchedulePlan,0),
	}

	go G_Scheduler.scheduleLoop()
	return
}

//push任务到调度器
func (sc *Scheduler) Push(jobEvent *common.JobEvent) {
	sc.jobEventChan <- jobEvent
}

//处理任务(维护任务表)
func (sc *Scheduler) handle(jobEvent *common.JobEvent) {
	var (
		jobSchedulePlan *SchedulePlan
		err error
	)
	switch jobEvent.Type {
	case common.JOB_EVENT_SAVE:
		if jobSchedulePlan,err = buidJobSchedulPlan(jobEvent.Job);err != nil {
			return
		}
		sc.jobEventPlanMap[jobEvent.Job.Name] = jobSchedulePlan
	case common.JOB_EVENT_DELETE:
		if _,ok := sc.jobEventPlanMap[jobEvent.Job.Name];ok{
			delete(sc.jobEventPlanMap,jobEvent.Job.Name)
		}
	case common.JOB_EVENT_KILL:
		//强杀任务必须在执行表中
		if jobSchedulePlan,ok := sc.jobExecMap[jobEvent.Job.Name];ok{
			//通过调用执行中的任务的cancelFunc()来中断任务执行
			fmt.Println("执行强杀任务")
			jobSchedulePlan.cancelFunc()
			//强杀任务后需要恢复不然永远无法执行。
			jobSchedulePlan.ctx,jobSchedulePlan.cancelFunc = context.WithCancel(context.TODO())
		}
	}
}

//根据当前任务构建下一次调度计划
func buidJobSchedulPlan(job *common.Job) (plan *SchedulePlan,err error) {
	var (
		expr *cronexpr.Expression
	)

	//解析Job的Cronb表达式
	if expr, err = cronexpr.Parse(job.CronExpr);err != nil {
		logger.Warn("Cron Parse Error errno:%d, err:%s",common.ERRNO_CRON_PARSE_FAILD,err.Error())
		return
	}
	ctx, cancelFunc := context.WithCancel(context.TODO())
	//生成调度计划
	plan = &SchedulePlan{
		Job:job,
		Expr:expr,
		NextTime:expr.Next(time.Now()),
		ctx:ctx,
		cancelFunc:cancelFunc,
	}


	return
}

//任务执行的时间如果超出其应该下次执行的时间时下次不再执行
func (sc *Scheduler) tryStartJob (plan *SchedulePlan) {
	//任务还在执行则跳过本次调度
	if _,ok := sc.jobExecMap[plan.Job.Name];ok {
		fmt.Println("任务还在执行中")
		return
	}

	//未执行则加入执行表中
	sc.jobExecMap[plan.Job.Name] = plan
	//执行cron命令
	G_Executor.ExecJob(plan)
}

package jobmgr

import (
	"go.etcd.io/etcd/clientv3"
	"time"
	"worker/config"
	"context"
	"helper/common"
	logger "github.com/shengkehua/xlog4go"
	"go.etcd.io/etcd/mvcc/mvccpb"
)

var (
	G_JobMgr *JobMgr
)

type JobMgr struct {
	Client *clientv3.Client
}

//初始化管理器
func Init() error{
	config := clientv3.Config{
		Endpoints:config.Cfg.JobMgr.Endpoints,
		DialTimeout: time.Duration(config.Cfg.JobMgr.TimeOut) * time.Millisecond,
	}

	if client,err := clientv3.New(config);err !=nil {
		return err
	}else {
		G_JobMgr = &JobMgr{
			Client:client,
		}
	}
	return nil
}

func (j *JobMgr) WatchJobs() (err error) {
	var (
		getResp *clientv3.GetResponse
		watchVersion int64
		watchChan clientv3.WatchChan
	)
	if getResp,err = j.Client.Get(context.TODO(),common.JOB_KEY_PREFIX,clientv3.WithPrefix());err != nil {
		logger.Error("WatchJob Error errno:%d, err:%s",common.ERRNO_ETCD_GET_FAILED,err.Error())
		return
	}

	//当前有哪些任务
	for _,v:= range getResp.Kvs {
		job := &common.Job{}
		if job,err = common.UnpackJob(v.Value);err !=nil {
			logger.Error("WatchJob Error errno:%d, err:%s",common.ERRNO_ETCD_GET_FAILED,err.Error())
			continue
		}
		jobEvent := common.BuildEvent(common.JOB_EVENT_SAVE,job)
		//防止写满阻塞流程
		go G_Scheduler.Push(jobEvent)

	}
	//从该version之后向后监听key的变化
	go func(){
		//从get版本之后开始监听
		watchVersion = getResp.Header.Revision + 1
		//启动监听
		watchChan = j.Client.Watch(context.TODO(),common.JOB_KEY_PREFIX,clientv3.WithRev(watchVersion),clientv3.WithPrefix())

		for resp := range watchChan {
			for _,res := range resp.Events {
				switch res.Type {
				case mvccpb.PUT: // 写key
				if job,err := common.UnpackJob(res.Kv.Value);err !=nil {
					//写入的val非法
					logger.Error("UnpackJob Error errno:%d, err:%s",common.ERRNO_JSON_UNMARSHAL_FAILED,err.Error())
					continue
				}else {
					jobEvent := common.BuildEvent(common.JOB_EVENT_SAVE,job)
					go G_Scheduler.Push(jobEvent)

				}

				case mvccpb.DELETE://删除key
				jobNmae := common.ExtractJobName(string(res.Kv.Key))
				job := &common.Job{
					Name:jobNmae,
				}
				jobEvent := common.BuildEvent(common.JOB_EVENT_DELETE,job)
				go G_Scheduler.Push(jobEvent)
				}
			}
		}
	}()
	return
}


func (j *JobMgr) NewJobLock(name string) (jobLock *JobLock) {
	return initJobLock(j.Client,name)
}

func (j *JobMgr) WatchKillJobs() (err error) {
	var (
		getResp *clientv3.GetResponse
		watchChan clientv3.WatchChan
	)
	if getResp,err = j.Client.Get(context.TODO(),common.JOB_KILL_PREFIX,clientv3.WithPrefix());err != nil {
		logger.Error("WatchJob Error errno:%d, err:%s",common.ERRNO_ETCD_GET_FAILED,err.Error())
		return
	}

	//当前有哪些任务
	for _,v:= range getResp.Kvs {
		job := &common.Job{}
		if job,err = common.UnpackJob(v.Value);err !=nil {
			logger.Error("WatchJob Error errno:%d, err:%s",common.ERRNO_ETCD_GET_FAILED,err.Error())
			continue
		}
		jobEvent := common.BuildEvent(common.JOB_EVENT_KILL,job)
		//防止写满阻塞流程
		go G_Scheduler.Push(jobEvent)

	}
	//从该version之后向后监听key的变化
	go func(){
		//从get版本之后开始监听
		//启动监听
		watchChan = j.Client.Watch(context.TODO(),common.JOB_KILL_PREFIX,clientv3.WithPrefix())

		for resp := range watchChan {
			for _,res := range resp.Events {
				switch res.Type {
				case mvccpb.PUT: // 写key(删除)
					jobName := common.ExtractKillJobName(string(res.Kv.Key))
					job := &common.Job{
						Name:jobName,
					}
					jobEvent := common.BuildEvent(common.JOB_EVENT_KILL,job)
					go G_Scheduler.Push(jobEvent)
				case mvccpb.DELETE://删除key,无影响
				}
			}
		}
	}()
	return
}

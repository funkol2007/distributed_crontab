package jobmgr

import (
	"go.etcd.io/etcd/clientv3"
	"time"
	"master/config"
	"helper/common"
	logger "github.com/shengkehua/xlog4go"
	"encoding/json"
	"context"
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

//保存job到etcd
func (j *JobMgr) SaveJob(job *common.Job) (oldJob *common.Job, err error) {
	var (
		jobKey    string
		jobVal    []byte
		putResp   *clientv3.PutResponse
	)
	//确定保存的key和val
	jobKey = common.JOB_KEY_PREFIX + job.Name
	if jobVal,err = json.Marshal(job);err !=nil {
		logger.Error("Parse Error errno:%d, err:%s",common.ERRNO_JSON_MARSHAL_FAILED,err.Error())
		return
	}

	//存入etcd
	if putResp,err = j.Client.Put(context.TODO(),jobKey,string(jobVal),clientv3.WithPrevKV());err !=nil {
		logger.Error("etcd put Error errno:%d, err:%s",common.ERRNO_ETCD_PUT_FAILED,err.Error())
		return
	}
	//如果是更新则返回旧值，否则为空
	if putResp.PrevKv != nil {
		if oldJob,err = common.UnpackJob(putResp.PrevKv.Value);err !=nil {
			logger.Error("UnpackJob Error errno:%d, err:%s",common.ERRNO_JSON_UNMARSHAL_FAILED,err.Error())
			//旧值解析错误打日志，不报错
			err = nil
		}
	}
	return
}

//删除etcd里的job
func (j *JobMgr) DeleteJob(name string) (oldJob *common.Job, err error) {
	var (
		jobKey string
		delResp *clientv3.DeleteResponse
	)
	jobKey = common.JOB_KEY_PREFIX + name

	//etcd删除key
	if delResp, err = j.Client.Delete(context.TODO(),jobKey,clientv3.WithPrevKV()); err != nil {
		logger.Error("Delete Error errno:%d, err:%s",common.ERRNO_ETCD_DELETE_FAILED,err.Error())
		return
	}

	//如果是删除一个不存在的key也没有影响，删除存在的可以就返回被删除的信息
	if len(delResp.PrevKvs) != 0  {
		if oldJob, err =common.UnpackJob(delResp.PrevKvs[0].Value);err !=nil {
			logger.Error("UnpackJob Error errno:%d, err:%s",common.ERRNO_JSON_UNMARSHAL_FAILED,err.Error())
			//旧值解析错误打日志，不报错
			err = nil
		}
	}
	return
}

//列出所有的任务
func (j *JobMgr) ListJob() (jobList []*common.Job,err error) {
	var (
		getResp *clientv3.GetResponse
	)
	if getResp, err = j.Client.Get(context.TODO(),common.JOB_KEY_PREFIX,clientv3.WithPrefix());err != nil {
		logger.Error("Get Error errno:%d, err:%s",common.ERRNO_ETCD_GET_FAILED,err.Error())
		return
	}

	//遍历所有的返回任务
	jobList = make([]*common.Job,0)
	for _,v := range getResp.Kvs {
		job := &common.Job{}
		if job, err = common.UnpackJob(v.Value);err !=nil {
			//解析失败打个日志继续
			logger.Error("UnpackJob Error errno:%d, err:%s",common.ERRNO_JSON_UNMARSHAL_FAILED,err.Error())
			err = nil
			continue
		}
		jobList = append(jobList,job)
	}
	return
}

//强制杀死任务
func (j *JobMgr) KillJob(name string) (err error) {
	//杀死任务就是向etcd写入杀死的key，这样worker会监听到然后执行杀死操作
	//TODO 这样做可能会出现worker执行杀死任务失败，然后写入的key已经过期的情况（出错或者宕机）。
	var (
		jobKey    string
		lease     *clientv3.LeaseGrantResponse
	)
	//确定保存的key和val
	jobKey = common.JOB_KILL_PREFIX + name
	//存入etcd,租约1s(不续租)
	if lease,  err  = j.Client.Grant(context.TODO(),1);err != nil {
		logger.Error("Grant Lease Error errno:%d, err :%s", common.ERRNO_ETCD_GRANT_LEASE_FAILED,err.Error())
		return
	}
	if _,err = j.Client.Put(context.TODO(),jobKey,"",clientv3.WithLease(lease.ID));err !=nil {
		logger.Error("etcd put Error errno:%d, err:%s",common.ERRNO_ETCD_PUT_FAILED,err.Error())
		return
	}

	return
}
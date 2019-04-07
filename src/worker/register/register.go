package register

import (
	"go.etcd.io/etcd/clientv3"
	logger "github.com/shengkehua/xlog4go"
	"helper/common"
	"time"
	"worker/config"
	"context"
)

//注册器
type Register struct {
	Client *clientv3.Client
}


var (
	G_Register *Register
)

//初始化注册器
func InitRegiter() (err error) {
	config := clientv3.Config{
		Endpoints:config.Cfg.JobMgr.Endpoints,
		DialTimeout: time.Duration(config.Cfg.JobMgr.TimeOut) * time.Millisecond,
	}

	if client,err := clientv3.New(config);err !=nil {
		return err
	}else {
		G_Register = &Register{
			Client:client,
		}
	}
	go G_Register.RegisterWorker()
	return nil
}

//注册服务
func (r *Register) RegisterWorker() {
	var (
		leaseResp *clientv3.LeaseGrantResponse
		leaseKeepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
		err error
		localIP string
		regKey string
		leaseId clientv3.LeaseID
	)

	//注册失败不断充实，防止网络抖动造成的短暂问题
	for {
		ctx, ctlFunc := context.WithCancel(context.TODO())
		//建立租约（5s）
		if leaseResp, err =  r.Client.Grant(ctx, 5); err != nil {
			logger.Error("Lease Grant Error errno:%d, err:%s", common.ERRNO_ETCD_GRANT_LEASE_FAILED, err.Error())
			goto ERR
		}

		//自动续租
		leaseId = leaseResp.ID
		if leaseKeepAliveChan, err = r.Client.KeepAlive(ctx, leaseId); err != nil {
			logger.Error("Lease Grant Error errno:%d, err:%s", common.ERRNO_ETCD_GRANT_LEASE_FAILED, err.Error())
			goto ERR
		}
		//写入/cron/workers/IP...作为注册
		if localIP,err = common.GetLocalIP();err !=nil {
			logger.Error("Get LocalIP Error errno:%d, err:%s", common.ERRNO_GET_LOCAL_IP_ERROR, err.Error())
			goto ERR
		}
		regKey = common.JOB_RGEISTER_PREFIX + localIP

		if _,err = r.Client.Put(ctx,regKey,"",clientv3.WithLease(leaseId)); err != nil {
			logger.Error("ETCD Put Error errno:%d, err:%s", common.ERRNO_ETCD_PUT_FAILED, err.Error())
			goto  ERR
		}
		//处理自动续租信号
		for {
				select {
				case r := <-leaseKeepAliveChan:
					if r == nil { //续租失败
						goto ERR
					}
				}
		}
		ERR:
			if ctlFunc != nil {
				time.Sleep(1 * time.Second)
				ctlFunc()
			}
	}

	return
}
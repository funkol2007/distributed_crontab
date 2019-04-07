package jobmgr

import (
	"context"
	"errors"
	logger "github.com/shengkehua/xlog4go"
	"go.etcd.io/etcd/clientv3"
	"helper/common"
)

type JobLock struct {
	kv         clientv3.KV
	lease      clientv3.Lease
	jobNmae    string
	leaseId    clientv3.LeaseID
	cancelFunc context.CancelFunc
	isLocked   bool
}

//初始化一把锁
func initJobLock(client *clientv3.Client, name string) (jobLock *JobLock) {
	jobLock = &JobLock{
		kv:      client.KV,
		lease:   client.Lease,
		jobNmae: name,
	}
	return
}

//尝试上乐观锁
func (j *JobLock) TryLock() (err error) {
	var (
		leaseResp          *clientv3.LeaseGrantResponse
		leaseKeepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
		txn                clientv3.Txn
		txnResp            *clientv3.TxnResponse
	)
	ctx, ctlFunc := context.WithCancel(context.TODO())
	//建立租约（1s）
	if leaseResp, err = j.lease.Grant(ctx, 5); err != nil {
		logger.Error("Lease Grant Error errno:%d, err:%s", common.ERRNO_ETCD_GRANT_LEASE_FAILED, err.Error())
		return
	}
	//自动续租
	leaseId := leaseResp.ID
	defer func() {
		//如果有错误，释放租约，取消自动续租
		if err != nil {
			ctlFunc()
			j.lease.Revoke(context.TODO(), leaseId)
		}
	}()
	if leaseKeepAliveChan, err = j.lease.KeepAlive(ctx, leaseId); err != nil {
		logger.Error("Lease Grant Error errno:%d, err:%s", common.ERRNO_ETCD_GRANT_LEASE_FAILED, err.Error())
		return
	}

	//处理自动续租信号
	go func() {
		for {
			select {
			case r := <-leaseKeepAliveChan:
				if r == nil { //续租失败
					return
				}
			}

		}
	}()
	//txn获取锁
	txn = j.kv.Txn(context.TODO())

	// 锁路径
	lockKey := common.JOB_LOCK_PREFIX + j.jobNmae

	// 5, 事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseId))).
		Else(clientv3.OpGet(lockKey))

	// 提交事务
	if txnResp, err = txn.Commit(); err != nil {
		return
	}

	// 6, 成功返回, 失败释放租约
	if !txnResp.Succeeded { // 锁被占用,获取失败
		err = errors.New("LockKey Got By Others")
		return
	}
	//抢锁成功,设置leaseId和取消函数
	j.leaseId = leaseId
	j.cancelFunc = ctlFunc
	j.isLocked = true
	return
}

//解锁
func (j *JobLock) UnLock() {
	if j.isLocked {
		j.cancelFunc()
		j.lease.Revoke(context.TODO(), j.leaseId)
	}
}

package httpserver

import (
	"net/http"
	"encoding/json"
	"helper/common"
	"master/jobmgr"
	"strconv"
	"master/logmgr"
	"master/workermgr"
)

//保存任务
//POST job = {"name":"job","command":"echo hello","cronExpr":"* * * * * *"}
func JobSaveHandler(resp http.ResponseWriter, req *http.Request) (response HttpResponser) {
	ret := &HttpResponse{
		ErrMsg:"OK",
		ErrNo:0,
	}
	var (
		errno int
		err error
		oldJob *common.Job
	)

	defer func(){
		if err != nil {
			ret.ErrMsg = err.Error()
		}
		ret.ErrNo = errno
	}()

	//解析Post表单
	if err = req.ParseForm();err != nil {
		errno  = common.ERRNO_PARSEPOST_FAILED
		return ret
	}
	//获取表单中的job对象并反序列化到结构体
	postJob := req.PostForm.Get(common.JOB)
	job := &common.Job{}
	if err = json.Unmarshal([]byte(postJob),job);err != nil {
		errno = common.ERRNO_JSON_UNMARSHAL_FAILED
		return ret
	}
	if oldJob,err = jobmgr.G_JobMgr.SaveJob(job);err != nil {
		errno = common.ERRNO_ETCD_PUT_FAILED
		return  ret
	}

	//把ret里的信息json话返回到resp里
	ret.Data = oldJob
	if _, err = ret.ResponseJson(resp);err != nil {
		errno = common.ERRNO_HTTP_RESPONSE_JSON_FAILED
	}

	return ret
}

//删除任务
func JobDeleteHandler(resp http.ResponseWriter, req *http.Request) (response HttpResponser) {
	ret := &HttpResponse{
		ErrMsg:"OK",
		ErrNo:0,
	}
	var (
		errno int
		err error
		oldJob *common.Job
	)

	defer func(){
		if err != nil {
			ret.ErrMsg = err.Error()
		}
		ret.ErrNo = errno
	}()

	postJobName := req.PostForm.Get(common.JOB_NAME)

	if oldJob, err = jobmgr.G_JobMgr.DeleteJob(postJobName);err !=nil {
		errno =  common.ERRNO_ETCD_DELETE_FAILED
		return ret
	}

	//把ret里的信息json返回到resp里
	ret.Data = oldJob
	if _, err = ret.ResponseJson(resp);err != nil {
		errno = common.ERRNO_HTTP_RESPONSE_JSON_FAILED
	}
	return ret
}

func JobListHandler(resp http.ResponseWriter, req *http.Request) (response HttpResponser) {
	ret := &HttpResponse{
		ErrMsg:"OK",
		ErrNo:0,
	}
	var (
		errno int
		err error
		jobList []*common.Job
	)

	defer func(){
		if err != nil {
			ret.ErrMsg = err.Error()
		}
		ret.ErrNo = errno
	}()

	if jobList,err = jobmgr.G_JobMgr.ListJob(); err != nil {
		errno = common.ERRNO_ETCD_GET_FAILED
		return ret
	}

	//把ret里的信息返回到resp
	ret.Data = jobList
	if _, err = ret.ResponseJson(resp);err != nil {
		errno = common.ERRNO_HTTP_RESPONSE_JSON_FAILED
	}
	return ret
}

func JobKillHandler(resp http.ResponseWriter, req *http.Request) (response HttpResponser) {
	ret := &HttpResponse{
		ErrMsg:"OK",
		ErrNo:0,
	}
	var (
		errno int
		err error
	)

	defer func(){
		if err != nil {
			ret.ErrMsg = err.Error()
		}
		ret.ErrNo = errno
	}()

	postJobName := req.PostForm.Get(common.JOB_NAME)

	if err = jobmgr.G_JobMgr.KillJob(postJobName);err !=nil {
		errno =  common.ERRNO_KILL_JOB_FAILED
		return ret
	}

	//把ret里的信息json返回到resp里
	if _, err = ret.ResponseJson(resp);err != nil {
		errno = common.ERRNO_HTTP_RESPONSE_JSON_FAILED
	}
	return ret
}

func JobLogHandler(resp http.ResponseWriter, req *http.Request) (response HttpResponser) {
	ret := &HttpResponse{
		ErrMsg:"OK",
		ErrNo:0,
	}
	var (
		errno int
		err error
		skipParam int
		limitParam int
	)

	defer func(){
		if err != nil {
			ret.ErrMsg = err.Error()
		}
		ret.ErrNo = errno
	}()

	jobName := req.Form.Get(common.JOB_NAME)
	if skipParam,err  = strconv.Atoi(req.Form.Get(common.JOB_SKIP));err != nil {
		//非法默认从第一条开始展示
		skipParam = 0
	}

	if limitParam,err = strconv.Atoi(req.Form.Get(common.JOB_LIMIT));err != nil {
		//非法默认10条
		limitParam = 10
	}
	//limitParam = 10
	if logArr,err := logmgr.G_LogMgr.ListLog(jobName,skipParam,limitParam);err !=nil {
		//日志获取失败
		errno = common.ERRNO_LOG_GET_FAILED
		return
	}else {
		ret.Data = logArr
	}

	//把ret里的信息json返回到resp里
	if _, err = ret.ResponseJson(resp);err != nil {
		errno = common.ERRNO_HTTP_RESPONSE_JSON_FAILED
	}
	return ret
}

func WorkerListHandler(resp http.ResponseWriter, req *http.Request) (response HttpResponser) {
	ret := &HttpResponse{
		ErrMsg:"OK",
		ErrNo:0,
	}
	var (
		errno int
		err error
		workerArr []string
	)

	defer func(){
		if err != nil {
			ret.ErrMsg = err.Error()
		}
		ret.ErrNo = errno
	}()

	if workerArr,err = workermgr.G_workerMgr.ListWorkers();err !=nil {
		errno =  common.ERRNO_GET_LOCAL_IP_ERROR
		return
	}
	ret.Data = workerArr
	//把ret里的信息json返回到resp里
	if _, err = ret.ResponseJson(resp);err != nil {
		errno = common.ERRNO_HTTP_RESPONSE_JSON_FAILED
	}
	return ret
}

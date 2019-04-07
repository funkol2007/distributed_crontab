package httpserver

import (
	"net/http"
	"strings"
	"time"
	"os"
	logger "github.com/shengkehua/xlog4go"
	"runtime/debug"
	"net"
	"syscall"
	"sync"
	"io"
	"encoding/json"
	"fmt"
	"helper/common"
	"master/config"
)

func init() {
	Uri2Handler                     = make(map[string]*HttpHandler)
	Uri2Handler["/job/save"]        = &HttpHandler{Name: "SaveJob", Handle: JobSaveHandler}
	Uri2Handler["/job/delete"]      = &HttpHandler{Name: "DeleteJob", Handle: JobDeleteHandler}
	Uri2Handler["/job/list"]        = &HttpHandler{Name: "ListJob", Handle: JobListHandler}
	Uri2Handler["/job/kill"]        = &HttpHandler{Name: "KillJob", Handle: JobKillHandler}
	Uri2Handler["/job/log"]         = &HttpHandler{Name: "KillJob", Handle: JobLogHandler}
	Uri2Handler["/worker/list"]         = &HttpHandler{Name: "KillJob", Handle: WorkerListHandler}
}

var (
	HttpListener net.Listener
	Uri2Handler  map[string]*HttpHandler
	onceHttp     sync.Once
	httpInstance *HttpServer
	staticDir = http.Dir(config.WebrootPath)
	StaticHandler = http.FileServer(staticDir)
)

type HttpResponser interface {
	//返回错误码, 用于监控
	ErrCode() int
	//返回内容给调用方
	ResponseJson(io.Writer) (int, error)
	//用于打印日志
	String() string
	//继承 error 接口
	Error() string
}

type HttpResponse struct {
	ErrNo  int    `json:"errno"`
	ErrMsg string `json:"errmsg"`
	LogId  string `json:"logid,omitempty"`
	Data   interface{}  `json"data"`
}

func (r *HttpResponse) ErrCode() int {
	return r.ErrNo
}

func (r *HttpResponse) ResponseJson(w io.Writer) (n int, err error) {
	var s []byte
	var s1 string
	s, err = json.Marshal(r)
	if err != nil {
		//unlikely, Marshal failed, 返回固定的信息
		//不要打印r, 小心无限递归
		logger.Error("json.Marshal err:%v", err)
		s1 = fmt.Sprintf("{\"errno\":%v,\"errmsg\":\"%v\",\"logid\":\"%v\"}", common.ERRNO_JSON_MARSHAL_FAILED, err, r.LogId)
	} else {
		s1 = string(s)
	}
	n, err = io.WriteString(w, s1)
	if err != nil {
		logger.Error("io.WriteString err:%v", err)
	}
	return
}

func (r *HttpResponse) Error() string {
	return fmt.Sprintf("errno=%v,errmsg=%v", r.ErrNo, r.ErrMsg)
}

func (r *HttpResponse) String() string {
	resJson, _ := json.Marshal(r)
	return string(resJson)
}

func doResponse(logid string, errno int, errmsg string, writer io.Writer) (r HttpResponser) {
	r = &HttpResponse{
		ErrNo:  errno,
		ErrMsg: errmsg,
		LogId:  logid,
	}
	_, err := r.ResponseJson(writer)
	if err != nil {
		logger.Error("doResponse err:%v", err)
	}
	return
}

type HttpHandler struct {
	Name      string
	Handle    func(w http.ResponseWriter, r *http.Request) HttpResponser
	waitGroup sync.WaitGroup
}

func (kh *HttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var clientLogId, clientSpanId string
	var logId int64
	var info map[string]interface{}
	var resp HttpResponser
	var errCode int

	tBegin := time.Now()
	kh.waitGroup.Add(1)

	defer func() {
		kh.waitGroup.Done()
		//耗时
		latency := time.Since(tBegin)
		if resp != nil {
			errCode = resp.ErrCode()
		} else {
			//unlikely
			errCode = -1
		}
		//捕捉panic
		if err := recover(); err != nil {
			errCode = common.ERRNO_PANIC
			logger.Error("LogId:%d HandleError# recover errno:%d stack:%s", logId, errCode, string(debug.Stack()))
			resp = doResponse(clientLogId, common.ERRNO_PANIC, "panic", w)
		}

		if errCode != 0 {
			logger.Error("%v [traceid:%v LogId:%d] errno:%d resp:%s", info["name"], clientLogId, logId, errCode, resp.String())
		}
		logger.Info("_com_request_out||funcname=%v||traceid=%v||spanid=%v||logId=%v||uri=%v||host=%v||remotAddr=%v||request=%v||response=%v||proc_time=%.2f", info["name"], clientLogId, clientSpanId, logId, info["url"], info["host"], info["remote"], info["param"], resp.String(), latency.Seconds()*1000)
	}()

	r.ParseForm()

	info = GetHttpRequestInfo(r)

	//log request in
	logger.Info("_com_request_in||funcname=%v||traceid=%v||spanid=%v||logId=%v||uri=%v||host=%v||remotAddr=%v||request=%v", info["name"], clientLogId, clientSpanId, logId, info["url"], info["host"], info["remote"], info["param"])

	resp = kh.Handle(w, r)
	return
}

func (kh *HttpHandler) Close() {
	kh.waitGroup.Wait()
}

func GetHttpRequestInfo(r *http.Request) (info map[string]interface{}) {
	info = make(map[string]interface{})
	info["clientLogId"] = r.Header.Get("didi-header-rid")
	info["url"] = r.URL.Path
	info["param"] = r.Form
	info["host"] = r.Host
	info["remote"] = r.RemoteAddr
	s1 := strings.Split(r.URL.Path, "/")
	if len(s1) > 0 {
		info["name"] = s1[len(s1)-1]
	} else {
		info["name"] = "NoBody"
	}
	info["now"] = time.Now()
	return
}

//HttpServer
type HttpServer struct {
	ServerMux   *http.ServeMux
	Uri2Handler map[string]*HttpHandler
	ListenPort  string
}

//单例模式
func GetHttpInstance() *HttpServer {
	onceHttp.Do(func() {
		if httpInstance == nil {
			httpInstance = &HttpServer{}
		}
	})
	return httpInstance
}

func (hs *HttpServer) AddHandler(uri string, handler *HttpHandler) error {
	hs.Uri2Handler[uri] = handler
	return nil
}

func (hs *HttpServer) Init(port string) error {
	hs.Uri2Handler = make(map[string]*HttpHandler)
	hs.ListenPort = port
	hs.ServerMux = http.NewServeMux()
	return nil
}

func (hs *HttpServer) Start() error {

	curPid := os.Getpid()
	go func(pid int) {
		osProcess := os.Process{Pid: pid}
		defer func() {
			if errRecover := recover(); errRecover != nil {
				logger.Error("abort, unknown error, errno:%d,errmsg:%v, stack:%s",
					common.ERRNO_PANIC, errRecover, string(debug.Stack()))
			}
		}()

		var err error
		HttpListener, err = net.Listen("tcp", ":"+hs.ListenPort)
		if err != nil {
			logger.Error("will_send_kill_cmd, tcp_listen_fail errmsg:%s", err.Error())
			osProcess.Signal(syscall.SIGINT)
			return
		}
		defer HttpListener.Close()

		//handler实现ServeHTTP接口就可以
		for uri, handler := range hs.Uri2Handler {
			hs.ServerMux.Handle(uri, handler)
		}
		//静态路由
		hs.ServerMux.Handle("/",http.StripPrefix("/",StaticHandler))

		server := http.Server{
			Handler:      hs.ServerMux,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		}
		err = server.Serve(HttpListener)
		if err != nil {
			logger.Warn("server_error errmsg:%s", err.Error())
		}
	}(curPid)
	return nil
}
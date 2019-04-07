package common

import (
	"runtime"
	"path"
	"strconv"
	"strings"
	"time"
	"math"
	"math/rand"
	"sync"
	"net"
	"errors"
)

var rand_gen = rand.New(rand.NewSource(time.Now().UnixNano()))
var lk sync.Mutex


// 获取本机网卡IP
func GetLocalIP() (ipv4 string, err error) {
	var (
		addrs []net.Addr
		addr net.Addr
		ipNet *net.IPNet // IP地址
		isIpNet bool
	)
	// 获取所有网卡
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	// 取第一个非lo的网卡IP
	for _, addr = range addrs {
		// 这个网络地址是IP地址: ipv4, ipv6
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			// 跳过IPV6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String()	// 192.168.1.1
				return
			}
		}
	}

	err = errors.New("没有IP网卡")
	return
}

// 提取worker的IP
func ExtractWorkerIP(regKey string) (string) {
	return strings.TrimPrefix(regKey, JOB_RGEISTER_PREFIX)
}

func CallerName() string {
	var pc uintptr
	var file string
	var line int
	var ok bool
	if pc, file, line, ok = runtime.Caller(1); !ok {
		return ""
	}
	name := runtime.FuncForPC(pc).Name()
	res := "[" + path.Base(file) + ":" + strconv.Itoa(line) + "]" + name
	tmp := strings.Split(name, ".")
	res = tmp[len(tmp)-1]
	return res
}

func RandInt() int {
	return rand_gen.Int()
}

func RandIntn(max int) int {
	lk.Lock()
	n := rand_gen.Intn(max)
	lk.Unlock()
	return n
}

func NowInS() int64 {
	return time.Now().Unix()
}

func NowInNs() int64 {
	return time.Now().UnixNano()
}

func NowInMs() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func Abs(x int32) int32 {
	switch {
	case x < 0:
		return -x
	case x == 0:
		return 0 // return correctly abs(-0)
	}
	return x
}
func Distance(flat float64, flng float64,
	tlat float64, tlng float64) (r int32) {
	distance := math.Sqrt((flat-tlat)*(flat-tlat) + (flng-tlng)*(flng-tlng))
	return int32(distance * 100000)
}

func String(data []byte, err error) string {
	if err == nil {
		return string(data)
	}
	return ""
}


func InArray(l []string, e string) bool {
	for _, v := range l {
		if v == e {
			return true
		}
	}

	return false
}

func InIntArray(l []int, e int) bool {
	for _, v := range l {
		if v == e {
			return true
		}
	}

	return false
}

func InInt32Array(l []int32, e int32) bool {
	for _, v := range l {
		if v == e {
			return true
		}
	}

	return false
}

func ArrInIntArray(arr1 []int, arr2 []int) bool {
	for _, v1 := range arr1 {
		for _, v2 := range arr2 {
			if v1 == v2 {
				return true
			}
		}
	}

	return false
}

func IsDaytime(hour int, daytime string) bool {
	if len(daytime) == 0 {
		return false
	}
	hourArr := strings.Split(daytime, ",")
	if len(hourArr) <= hour {
		return false
	}
	if hourArr[hour] == "1" {
		return true
	} else {
		return false
	}
}

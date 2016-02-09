/*
* @Author: detailyang
* @Date:   2016-02-09 15:13:37
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-09 21:58:05
 */

package httpserver

import (
	"dialer"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/glog"
	"io"
	"net"
	"net/http"
	"time"
	"utils"
)

type HttpServer struct {
	local      string
	redispool  *redis.Pool
	listener   net.Listener
	dialerpool *dialer.DialerPool
}

func NewHttpServer(local string, redispool *redis.Pool, dialerpool *dialer.DialerPool) *HttpServer {
	return &HttpServer{
		local:      local,
		redispool:  redispool,
		dialerpool: dialerpool,
	}
}

func (self *HttpServer) Run() {
	glog.V(0).Infoln("http server listen on ", self.local)
	err := http.ListenAndServe(self.local, http.HandlerFunc(self.handle))
	if err != nil {
		glog.Infoln("listen get error ", err)
	}
}

func (self *HttpServer) request(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		glog.Errorln("split remote address error ", err)
		return
	}
	cache := self.redispool.Get()
	if cache == nil {
		glog.Errorln("get empty from redis pool ")
		return
	}
	upstreamaddress, err := redis.String(cache.Do("GET", host))
	if err != nil && upstreamaddress != "" {
		glog.Errorln("get empty from redis key ", "abcd")
	}
	r.RequestURI = ""
	dialer := self.dialerpool.Get(upstreamaddress)
	if dialer == nil {
		glog.Errorln("get empty from dialer pool ")
		return
	}
	transport := &http.Transport{
		Proxy:               nil,
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	client := &http.Client{Transport: transport}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return utils.ErrHttpRedirect
	}
	if r.URL.Scheme == "" {
		r.URL.Scheme = "http"
	}
	r.Host = upstreamaddress
	r.URL.Host = r.Host
	resp, err := client.Do(r)
	if err != nil && resp == nil {
		http.Error(w, "Error contacting backend server.", 500)
		glog.Errorf("Error connect backend %s: %v", r.RequestURI, err)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}

	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		glog.V(0).Infoln("io copy error ", err)
	}
}

func (self *HttpServer) handle(w http.ResponseWriter, r *http.Request) {
	if ok := net.ParseIP(r.Host); ok != nil {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("only support domain access"))
		return
	}
	self.request(w, r)
}

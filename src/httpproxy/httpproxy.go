/*
* @Author: detailyang
* @Date:   2016-02-09 15:13:37
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-09 22:05:11
 */

package httpproxy

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

type HttpProxy struct {
	local      string
	redispool  *redis.Pool
	dialerpool *dialer.DialerPool
	listener   net.Listener
}

func NewHttpProxy(local string, redispool *redis.Pool, dialerpool *dialer.DialerPool) *HttpProxy {
	return &HttpProxy{
		local:      local,
		dialerpool: dialerpool,
		redispool:  redispool,
	}
}

func (self *HttpProxy) Run() {
	glog.V(0).Infoln("http proxy listen on ", self.local)
	err := http.ListenAndServe(self.local, http.HandlerFunc(self.handle))
	if err != nil {
		glog.Infoln("listen get error ", err)
	}
}

func (self *HttpProxy) connect(w http.ResponseWriter, r *http.Request) {
	glog.V(0).Infof("client %s ready httpproxy to %s", r.RemoteAddr, r.RequestURI)
	serverConn, err := self.dialerpool.Dial("tcp", r.RequestURI)
	if err != nil {
		http.Error(w, "Error contacting backend server.", 500)
		glog.Errorf("Error connect backend %s: %v", r.RequestURI, err)
		return
	}
	defer serverConn.Close()
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "sorry, i cannot be hijacker?", 500)
		return
	}
	clientConn, _, err := hj.Hijack()
	if err != nil {
		glog.Infoln("Hijack error: %v", err)
		return
	}
	defer clientConn.Close()
	_, err = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		glog.Errorln("write error ", err)
		return
	}
	glog.V(0).Infof("success write HTTP/1.1 200 Connection Established to client %s", r.RemoteAddr)

	err = utils.TcpPipe(serverConn, clientConn)
	if err != nil {
		glog.Errorln("tcp pipe error ", err)
	}
}

func (self *HttpProxy) request(w http.ResponseWriter, r *http.Request) {
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
	r.RequestURI = ""
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

func (self *HttpProxy) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == "CONNECT" {
		self.connect(w, r)
		return
	}
	self.request(w, r)
}

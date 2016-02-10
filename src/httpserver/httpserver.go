/*
* @Author: detailyang
* @Date:   2016-02-09 15:13:37
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-10 19:26:59
 */

package httpserver

import (
	"dialer"
	"github.com/golang/glog"
	"github.com/spf13/viper"
	"io"
	"net"
	"net/http"
	"time"
	"utils"
)

type HttpServer struct {
	utils.ACL
	name       string
	local      string
	listener   net.Listener
	dialerpool *dialer.DialerPool
	vip        *viper.Viper
}

func NewHttpServer(name, local string, dialerpool *dialer.DialerPool, vip *viper.Viper) *HttpServer {
	return &HttpServer{
		name:       name,
		local:      local,
		dialerpool: dialerpool,
		vip:        vip,
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
	whitelisthosts := self.vip.GetString("whitelisthosts")
	if self.MatchHost(whitelisthosts, r.Host) == false {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("sorry '" + r.Host + "' is not in whitelist"))
		return
	}
	upstreams := self.vip.GetStringMapString("httpservers.hs1.upstreams")
	upstream, ok := upstreams[r.Host]
	if ok == false {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("sorry '" + r.Host + "' have not set upstream"))
		return
	}
	r.RequestURI = ""
	dialer := self.dialerpool.Get(upstream)
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
	r.Host = upstream
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

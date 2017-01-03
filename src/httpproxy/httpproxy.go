/*
* @Author: detailyang
* @Date:   2016-02-09 15:13:37
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-15 10:57:17
 */

package httpproxy

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

type HttpProxy struct {
	utils.ACL
	name       string
	local      string
	dialerpool *dialer.DialerPool
	listener   net.Listener
	vip        *viper.Viper
}

func NewHttpProxy(name, local string, dialerpool *dialer.DialerPool, vip *viper.Viper) *HttpProxy {
	return &HttpProxy{
		name:       name,
		local:      local,
		dialerpool: dialerpool,
		vip:        vip,
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
	whitelisthosts := self.vip.GetStringSlice("whitelisthosts")
	if self.MatchHost(whitelisthosts, r.RequestURI) == false {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("sorry '" + r.RequestURI + "' is not in whitelist"))
		return
	}
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

	err = utils.TcpPipe(serverConn, clientConn)
	if err != nil {
		glog.Errorln("tcp pipe error ", err)
	}
}

func (self *HttpProxy) request(w http.ResponseWriter, r *http.Request) {
	whitelisthosts := self.vip.GetStringSlice("whitelisthosts")
	if self.MatchHost(whitelisthosts, r.Host) == false {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("sorry '" + r.Host + "' is not in whitelist"))
		return
	}
	dialer := self.dialerpool.Get(r.Host)
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

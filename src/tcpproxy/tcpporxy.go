/*
* @Author: detailyang
* @Date:   2016-02-09 01:32:38
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-09 21:22:44
 */

package tcpproxy

import (
	"dialer"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/glog"
	"net"
	"utils"
)

type TcpProxy struct {
	local      string
	redispool  *redis.Pool
	dialerpool *dialer.DialerPool
	listener   net.Listener
}

func NewTcpProxy(local string, redispool *redis.Pool, dialerpool *dialer.DialerPool) *TcpProxy {
	l, err := net.Listen("tcp", local)
	if err != nil {
		glog.Infoln("listen error: ", err)
	}
	glog.V(0).Infoln("tcp proxy listen on ", local)
	return &TcpProxy{
		local:      local,
		redispool:  redispool,
		listener:   l,
		dialerpool: dialerpool,
	}
}

func (self *TcpProxy) Run() {
	for {
		clientConn, err := self.listener.Accept()
		if err != nil {
			glog.Infoln("receive accept queue error ", err)
		}
		glog.Infoln("receive connection from ", clientConn.RemoteAddr().String())

		go self.handle(clientConn)
	}
}

func (self *TcpProxy) handle(clientConn net.Conn) {
	host, _, err := net.SplitHostPort(clientConn.RemoteAddr().String())
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
	upstreamConn, err := self.dialerpool.Dial("tcp", upstreamaddress)
	if err != nil {
		glog.Errorf("downstream '%s' dial upstream '%s' error :%s", clientConn.RemoteAddr().String(), upstreamaddress, err.Error())
		clientConn.Write([]byte("upstream is wrong: " + err.Error()))
		clientConn.Close()
		return
	}
	utils.TcpPipe(upstreamConn, clientConn)
}

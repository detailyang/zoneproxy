/*
* @Author: detailyang
* @Date:   2016-02-09 01:32:38
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-10 19:25:06
 */

package tcpproxy

import (
	"dialer"
	"github.com/golang/glog"
	"github.com/spf13/viper"
	"net"
	"utils"
)

type TcpProxy struct {
	utils.ACL
	name       string
	local      string
	dialerpool *dialer.DialerPool
	listener   net.Listener
	vip        *viper.Viper
}

func NewTcpProxy(name, local string, dialerpool *dialer.DialerPool, vip *viper.Viper) *TcpProxy {
	l, err := net.Listen("tcp", local)
	if err != nil {
		glog.Fatal("listen error: ", err)
	}
	glog.V(0).Infoln("tcp proxy listen on ", local)
	return &TcpProxy{
		name:       name,
		local:      local,
		listener:   l,
		dialerpool: dialerpool,
		vip:        vip,
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
	upstream := self.vip.GetString("tcpproxys." + self.name + ".upstream")
	whitelisthosts := self.vip.GetString("whitelisthosts")
	if self.MatchHost(whitelisthosts, upstream) == false {
		clientConn.Write([]byte("sorry '" + upstream + "' is not in whitelist"))
		return
	}
	upstreamConn, err := self.dialerpool.Dial("tcp", upstream)
	if err != nil {
		glog.Errorf("downstream '%s' dial upstream '%s' error :%s",
			clientConn.RemoteAddr().String(), upstream, err.Error())
		clientConn.Write([]byte("upstream is wrong: " + err.Error()))
		return
	}
	utils.TcpPipe(upstreamConn, clientConn)
}

/*
* @Author: detailyang
* @Date:   2016-02-09 20:52:52
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-10 19:16:16
 */

package dialer

import (
	"github.com/golang/glog"
	"golang.org/x/net/proxy"
	"net"
	"time"
	"utils"
)

type Dial struct {
	name   string
	ipnet  *net.IPNet
	dialer proxy.Dialer
}

type DialerPool struct {
	pool map[string]*Dial
}

func NewDialer(name, address, username, password, cidr string) *Dial {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		glog.Errorf("parse %s cidr error %s", err.Error())
		return nil
	}
	var auth *proxy.Auth
	if username != "" && password != "" {
		auth = &proxy.Auth{User: username, Password: password}
	} else {
		auth = nil
	}
	dialer, err := proxy.SOCKS5("tcp", address,
		auth,
		&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		},
	)
	if err != nil {
		glog.Errorln("connect socks5 proxy error ", err)
		return nil
	}

	return &Dial{
		name:   name,
		ipnet:  ipnet,
		dialer: dialer,
	}
}

func NewDialerPool() *DialerPool {
	pool := make(map[string]*Dial)
	return &DialerPool{
		pool: pool,
	}
}

func (self *DialerPool) Dial(network, hostport string) (net.Conn, error) {
	host := utils.GetHost(hostport)
	//o(n) maybe a little slow:)
	for _, dial := range self.pool {
		if dial.ipnet.Contains(net.ParseIP(host)) == false {
			continue
		}
		return dial.dialer.Dial(network, hostport)
	}

	glog.Infoln("cannot found any ipnet for ", host)
	for _, dial := range self.pool {
		return dial.dialer.Dial(network, hostport)
	}

	return nil, utils.ErrEmptyDialerPool
}

func (self *DialerPool) Get(hostport string) proxy.Dialer {
	host := utils.GetHost(hostport)
	if host == "" {
		for _, dial := range self.pool {
			return dial.dialer
		}
	}
	for _, dial := range self.pool {
		if dial.ipnet.Contains(net.ParseIP(host)) == false {
			continue
		}
		return dial.dialer
	}

	glog.Infoln("cannot found any ipnet for ", host)
	for _, dial := range self.pool {
		return dial.dialer
	}

	return nil
}

func (self *DialerPool) Add(name string, dial *Dial) {
	self.pool[name] = dial
}

func (self *DialerPool) AddByZones(zones map[string]interface{}) {
	for name, value := range zones {
		socks5 := value.(map[string]interface{})["socks5"]
		address := socks5.(map[string]interface{})["address"].(string)
		username := socks5.(map[string]interface{})["username"].(string)
		password := socks5.(map[string]interface{})["password"].(string)
		cidr := value.(map[string]interface{})["cidr"].(string)
		dialer := NewDialer(name, address, username, password, cidr)
		if dialer == nil {
			glog.Infof("new dialer error %s %s %s", name, address, cidr)
			continue
		}
		self.Add(name, dialer)
	}
}

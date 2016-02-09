/*
* @Author: detailyang
* @Date:   2016-02-09 02:36:31
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-09 21:55:01
 */

package utils

import (
	"github.com/golang/glog"
	"io"
	"net"
)

func TcpPipe(dst, src net.Conn) error {
	defer func() {
		_dst := dst.(*net.TCPConn)
		_src := src.(*net.TCPConn)
		_src.CloseRead()
		_dst.CloseWrite()
	}()
	done := make(chan error, 1)

	cp := func(r, w net.Conn) {
		n, err := io.Copy(r, w)
		done <- err
		if err != nil {
			glog.Errorln("io copy get error ", err)
		} else {
			glog.V(0).Infof("copied %d bytes from %s to %s", n, r.RemoteAddr(), w.RemoteAddr())
		}
	}

	go cp(dst, src)
	go cp(src, dst)

	err1 := <-done
	err2 := <-done
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}

func GetHost(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		glog.Infof("split remote address %s error ", hostport, err)
		return ""
	}
	return host
}

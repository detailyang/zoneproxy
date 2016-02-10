/*
* @Author: detailyang
* @Date:   2016-02-09 02:36:31
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-10 19:24:50
 */

package utils

import (
	"github.com/golang/glog"
	"io"
	"net"
)

func TcpPipe(dst, src net.Conn) error {
	done := make(chan error, 1)

	cp := func(r, w net.Conn) {
		defer func() {
			_r := r.(*net.TCPConn)
			_w := w.(*net.TCPConn)
			_r.CloseRead()
			_w.CloseWrite()
		}()
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
		return ""
	}
	return host
}

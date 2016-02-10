/*
* @Author: detailyang
* @Date:   2016-02-10 04:17:38
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-10 19:21:48
 */

package main

import (
	"flag"
	"github.com/armon/go-socks5"
	"github.com/golang/glog"
	"github.com/spf13/viper"
	"os"
	"os/Signal"
	"sync"
	"syscall"
)

func main() {
	var wg sync.WaitGroup
	// Create a SOCKS5 server
	var config string
	flag.StringVar(&config, "config", "", "config file")
	flag.Parse()
	if config == "" {
		glog.Fatalln("config file cannot be null")
	}
	viper.SetConfigFile(config)
	viper.SetConfigType("json")
	err := viper.ReadInConfig()
	if err != nil {
		glog.Fatalln("Fatal error config file ", err)
	}
	username := viper.GetString("username")
	password := viper.GetString("password")
	creds := socks5.StaticCredentials{}
	creds[username] = password
	cator := socks5.UserPassAuthenticator{Credentials: creds}
	conf := &socks5.Config{
		AuthMethods: []socks5.Authenticator{cator},
		Credentials: creds,
	}
	server, err := socks5.New(conf)
	if err != nil {
		glog.Infoln("new socks5 config error ", err)
	}

	address := viper.GetString("address")
	glog.Infoln("listen to address ", address)
	wg.Add(1)
	go func() {
		if err := server.ListenAndServe("tcp", address); err != nil {
			glog.Errorln(err)
		}
		wg.Done()
	}()

	// signal hup
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	wg.Add(1)
	go func() {
		for sig := range c {
			switch sig {
			case syscall.SIGHUP:
				glog.Infoln("got hup signal")
			default:
				glog.Infoln("not ready to process ", sig.String())
			}
		}
		wg.Done()
	}()

	wg.Wait()
}

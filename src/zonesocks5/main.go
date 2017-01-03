/*
* @Author: detailyang
* @Date:   2016-02-10 04:17:38
* @Last Modified by:   detailyang
* @Last Modified time: 2016-02-15 19:17:56
 */

package main

import (
	"flag"
	"github.com/armon/go-socks5"
	"github.com/golang/glog"
	"github.com/sevlyar/go-daemon"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"utils"
)

func main() {
	var config string
	var nodaemon bool

	signal := flag.String("s", "", "send signal to daemon")

	flag.StringVar(&config, "config", "", "config file")
	flag.BoolVar(&nodaemon, "nodaemon", false, "dont daemonize")
	flag.Parse()
	if config == "" {
		glog.Fatalln("config file cannot be null")
	}
	v := viper.New()
	v.SetConfigFile(config)
	v.SetConfigType("json")
	err := v.ReadInConfig()
	if err != nil {
		glog.Fatalln("Fatal error config file ", err)
	}

	signaldispatcher := func(sig os.Signal) error {
		return singalhandler(sig, v)
	}

	if nodaemon == true {
		log.Println("no daemonize to start up")
		zonesocks5(v)
		return
	}

	// Define command: command-line arg, system signal and handler
	daemon.AddCommand(daemon.StringFlag(signal, "stop"), syscall.SIGTERM, signaldispatcher)
	daemon.AddCommand(daemon.StringFlag(signal, "reload"), syscall.SIGHUP, signaldispatcher)
	flag.Parse()

	pidfile := v.GetString("pidfile")
	pidfileperms := v.GetString("pidfileperm")
	pidfileperm, err := strconv.ParseInt(pidfileperms, 8, 32)
	if err != nil {
		log.Println("parse pidfileperm error ", err)
	}
	logfile := v.GetString("logfile")
	logfileperms := v.GetString("logfileperm")
	logfileperm, err := strconv.ParseInt(logfileperms, 8, 32)
	if err != nil {
		log.Println("parse pidfileperm error ", err)
	}
	workdir := v.GetString("workdir")
	umasks := v.GetString("umask")
	umask, err := strconv.ParseInt(umasks, 8, 32)
	if err != nil {
		log.Println("parse umask error ", err)
	}

	dmn := &daemon.Context{
		PidFileName: pidfile,
		PidFilePerm: os.FileMode(pidfileperm),
		LogFileName: logfile,
		LogFilePerm: os.FileMode(logfileperm),
		WorkDir:     workdir,
		Umask:       int(umask),
	}

	// Send commands if needed
	if len(daemon.ActiveFlags()) > 0 {
		d, err := dmn.Search()
		if err != nil {
			log.Fatalln("Unable send signal to the daemon:", err)
		}
		err = daemon.SendCommands(d)
		if err != nil {
			log.Println(err)
		}
		return
	}

	// Process daemon operations - send signal if present flag or daemonize
	child, err := dmn.Reborn()
	if err != nil {
		log.Fatalln(err)
	}
	if child != nil {
		log.Println("daemonize to start up")
		log.Println("daemonize success")
		return
	}
	defer dmn.Release()

	// Run main operation
	go func() {
		zonesocks5(v)
	}()

	err = daemon.ServeSignals()
	if err != nil {
		log.Println("startup error:", err)
	}
}

func zonesocks5(v *viper.Viper) {
	var wg sync.WaitGroup
	// Create a SOCKS5 server
	username := v.GetString("username")
	password := v.GetString("password")
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

	address := v.GetString("address")
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

func singalhandler(sig os.Signal, v *viper.Viper) error {
	switch sig {
	case syscall.SIGHUP:
		log.Println("got hup signal, now reloading conf")
		err := v.ReadInConfig()
		if err != nil {
			glog.Infoln("Fatal error config file ", err)
			return utils.ErrReadConfig
		}
	case syscall.SIGTERM:
		log.Println("receive SIGTERM, exit")
		//maybe graceful stop is better:)
		os.Exit(0)
	default:
		log.Println(sig)
		glog.Infoln("not ready to process ", sig.String())
	}

	return nil
}

package main

import (
	"dialer"
	"flag"
	"github.com/golang/glog"
	"github.com/sevlyar/go-daemon"
	"github.com/spf13/viper"
	"httpproxy"
	"httpserver"
	"log"
	"os"
	"strconv"
	"sync"
	"syscall"
	"tcpproxy"
	"utils"
)

func main() {
	var config string
	var nodaemon bool

	dp := dialer.NewDialerPool()
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
		return singalhandler(sig, v, dp)
	}

	if nodaemon == true {
		log.Println("no daemonize to start up")
		zoneproxy(v, dp)
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
		zoneproxy(v, dp)
	}()

	err = daemon.ServeSignals()
	if err != nil {
		log.Println("startup error:", err)
	}
}

func zoneproxy(v *viper.Viper, dp *dialer.DialerPool) {
	var wg sync.WaitGroup

	zones := v.GetStringMap("zones")
	dp.AddByZones(zones)

	tcpproxys := v.GetStringMap("tcpproxys")
	for name, _ := range tcpproxys {
		address := v.GetString("tcpproxys." + name + ".address")
		if address == "" {
			glog.Fatalln("tcpproxys." + name + ".address must be string")
		}
		tp := tcpproxy.NewTcpProxy(name, address, dp, v)
		wg.Add(1)
		go func() {
			tp.Run()
			wg.Done()
		}()
	}

	httpproxys := v.GetStringMap("httpproxys")
	for name, _ := range httpproxys {
		address := v.GetString("httpproxys." + name + ".address")
		if address == "" {
			glog.Fatalln("httpproxys." + name + ".address must be string")
		}
		hp := httpproxy.NewHttpProxy(name, address, dp, v)
		wg.Add(1)
		go func() {
			hp.Run()
			wg.Done()
		}()
	}

	httpservers := v.GetStringMap("httpservers")
	for name, _ := range httpservers {
		address := v.GetString("httpservers." + name + ".address")
		if address == "" {
			glog.Fatalln("httpservers." + name + ".address must be string")
		}
		hs := httpserver.NewHttpServer(name, address, dp, v)
		wg.Add(1)
		go func() {
			hs.Run()
			wg.Done()
		}()
	}

	wg.Wait()
	glog.Flush()
}

func singalhandler(sig os.Signal, v *viper.Viper, dp *dialer.DialerPool) error {
	switch sig {
	case syscall.SIGHUP:
		log.Println("HUP")
		glog.Infof("got hup signal, now reloading conf\n", sig.String())
		err := v.ReadInConfig()
		if err != nil {
			glog.Infoln("Fatal error config file ", err)
			return utils.ErrReadConfig
		}
		zones := v.GetStringMap("zones")
		dp.AddByZones(zones)
	case syscall.SIGTERM:
		glog.Infoln("receive SIGTERM, exit")
		//maybe graceful stop is better:)
		os.Exit(0)
	default:
		log.Println(sig)
		glog.Infoln("not ready to process ", sig.String())
	}

	return nil
}

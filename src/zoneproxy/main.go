package main

import (
	"dialer"
	"flag"
	"github.com/golang/glog"
	"github.com/spf13/viper"
	"httpproxy"
	"httpserver"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"tcpproxy"
)

func main() {
	var wg sync.WaitGroup
	var config string

	flag.StringVar(&config, "config", "", "config file")
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

	zones := v.GetStringMap("zones")
	dp := dialer.NewDialerPool()
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

	// signal hup
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)
	wg.Add(1)
	go func() {
		for sig := range c {
			switch sig {
			case syscall.SIGHUP:
				glog.Infof("got hup signal, now reloading conf\n", sig.String())
				v.ReadInConfig()
				if err != nil {
					glog.Infoln("Fatal error config file ", err)
				}
				zones := v.GetStringMap("zones")
				dp.AddByZones(zones)
			default:
				glog.Infoln("not ready to process ", sig.String())
			}
		}
		wg.Done()
	}()

	wg.Wait()
	glog.Flush()
}

package main

import (
	"dialer"
	"flag"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/glog"
	"github.com/spf13/viper"
	"httpproxy"
	"httpserver"
	"sync"
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
	viper.SetConfigFile(config)
	viper.SetConfigType("json")
	err := viper.ReadInConfig()
	if err != nil {
		glog.Fatalln("Fatal error config file ", err)
	}
	r := viper.GetStringMap("redis")

	// socks5 proxy init
	zones := viper.GetStringMap("zones")
	dp := dialer.NewDialerPool()

	// TODO: deal with config better
	for name, value := range zones {
		socks5 := value.(map[string]interface{})["socks5"]
		address := socks5.(map[string]interface{})["address"].(string)
		username := socks5.(map[string]interface{})["username"].(string)
		password := socks5.(map[string]interface{})["password"].(string)
		cidr := value.(map[string]interface{})["cidr"].(string)
		dialer := dialer.NewDialer(name, address, username, password, cidr)
		if dialer == nil {
			glog.Infof("new dialer error %s %s %s", name, address, cidr)
			continue
		}
		dp.Add(name, dialer)
	}

	// redis init
	raddress := r["address"].(string)
	if raddress == "" {
		glog.Fatalln("should set redis address", err)
	}
	redispool := &redis.Pool{
		MaxIdle:   80,
		MaxActive: 10000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", raddress)
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}

	// should check config:)
	tp := tcpproxy.NewTcpProxy(viper.GetString("tcpproxy.address"), redispool, dp)
	hp := httpproxy.NewHttpProxy(viper.GetString("httpproxy.address"), redispool, dp)
	hs := httpserver.NewHttpServer(viper.GetString("httpserver.address"), redispool, dp)

	wg.Add(3)
	go func() {
		tp.Run()
		wg.Done()
	}()
	go func() {
		hp.Run()
		wg.Done()
	}()
	go func() {
		hs.Run()
		wg.Done()
	}()
	wg.Wait()
	glog.Flush()
}

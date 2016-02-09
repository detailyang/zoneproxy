GOPATH := ${GOPATH}:$(shell pwd)

all:
	@GOPATH=$(GOPATH) go install proxy
install:
	#maybe we should find other better package management system to shoot go get:(
	go get github.com/golang/glog
	go get golang.org/x/net/proxy
	go get github.com/spf13/viper
	go get github.com/garyburd/redigo/redis
clean:
	@rm -fr bin pkg
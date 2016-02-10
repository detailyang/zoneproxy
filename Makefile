GOPATH := ${GOPATH}:$(shell pwd)

hijack:
	@echo "hello jack"
install:
	#maybe we should find other better package management system to shoot go get:(
	go get github.com/golang/glog
	go get golang.org/x/net/proxy
	go get github.com/spf13/viper
	go get github.com/armon/go-socks5
build:
	@GOPATH=$(GOPATH) go install zoneproxy
	@GOPATH=$(GOPATH) go install socks5
buildarch:
	for arch in arm 386 amd64; do for os in linux darwin freebsd; do\
		go build -o bin/zoneproxy-$$os-$$arch zoneproxy &&\
		go build -o bin/socks5-$$os-$$arch socks5 &&\
	 	zip -r zoneproxy-$$os-$$arch.zip bin/zoneproxy-$$os-$$arch conf README.md; \
	done done
test:
	#fake test, we should add test :(
	@GOPATH=$(GOPATH) go test dialer
	@GOPATH=$(GOPATH) go test httpproxy
	@GOPATH=$(GOPATH) go test httpserver
	@GOPATH=$(GOPATH) go test tcpproxy
	@GOPATH=$(GOPATH) go test utils
clean:
	@rm -fr bin pkg

GOPATH := ${GOPATH}:$(shell pwd)

hijack:
	@echo "hello jack"
install:
	#maybe we should find other better package management system to shoot go get:(
	go get github.com/golang/glog
	go get golang.org/x/net/proxy
	go get github.com/spf13/viper
	go get github.com/armon/go-socks5
	go get github.com/sevlyar/go-daemon
build:
	@GOPATH=$(GOPATH) go install zoneproxy
	@GOPATH=$(GOPATH) go install zonesocks5
buildarch:
	git rev-parse HEAD > version
	for arch in amd64 386 arm; do for os in darwin linux freebsd; do\
		rm -rf zoneproxy &&\
		GOARCH=$$arch GOOS=$$os go build -o bin/zoneproxy zoneproxy &&\
		GOARCH=$$arch GOOS=$$os go build -o bin/socks5 socks5 &&\
		mkdir zoneproxy && cp -r bin zoneproxy/bin && cp -r conf zoneproxy/conf && \
		cp README.md zoneproxy && cp version zoneproxy && \
	 	tar -zcf zoneproxy-$$os-$$arch.tar.gz zoneproxy; \
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

GOPATH := ${GOPATH}:$(shell pwd)

all:
	@GOPATH=$(GOPATH) go install proxy
save:
	@GOPATH=$(GOPATH) godep save
clean:
	@rm -fr bin pkg
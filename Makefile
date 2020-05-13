
VERSION?=v0.0.1
BUILDFLAGS=CGO_ENABLED=0 GOOS=linux GOARCH=amd64 
BINARY=leveldbraft

.PHONY:build
build:
	${BUILDFLAGS} go build ./cmd/leveldbraft/
 
run: build
	./${BINARY}
	
clean:
	@rm ${BINARY}

docker: build
	docker build -t 00arthur00/${BINARY}:${VERSION} .
	docker tag 00arthur00/${BINARY}:${VERSION} 00arthur00/${BINARY}:latest
	

dockerpush: docker
	docker push 00arthur00/${BINARY}:${VERSION}
	docker push 00arthur00/${BINARY}:latest
	
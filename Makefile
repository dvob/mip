#
# mip Makefile
#

BINARY=mip
WIN_BINARY=mip-win64.exe

VERSION=`git describe --tags --dirty --always`
GIT_COMMIT=`git rev-parse HEAD`
BUILD_DATE=`date +%FT%T%z`

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.commit=${GIT_COMMIT} -X main.date=${BUILD_DATE}"

build:
	go build -o ${BINARY} ${LDFLAGS} github.com/dsbrng25b/mip/cmd/mip
	GOOS=windows GOARCH=amd64 go build -o ${WIN_BINARY} ${LDFLAGS} github.com/dsbrng25b/mip/cmd/mip

clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
	if [ -f ${WIN_BINARY} ] ; then rm ${WIN_BINARY} ; fi

.PHONY: clean

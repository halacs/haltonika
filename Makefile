DIST=dist/
APPNAME=haltonika

all: clean build

env:
	mkdir -p ${DIST}

clean:
	rm -rf ${DIST}${APPNAME}

build: env
	CGO_ENABLED=0 go build -v -o ${DIST}${APPNAME} .

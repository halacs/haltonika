DIST=dist
APPNAME=haltonika

GOLANGCILINT_VERSION=v1.61.0
GOSEC_VERSION=v2.21.4
VULNCHECK_VERSION=latest

TEMP_DIR=temp
DEPS_DIR=${DIST}
SRC_DIR_DEBIAN=package/debian
ARCH=amd64
VERSION=1.1.2
# Revision of the package file
REVISION=1.2.0
# Location of package files
DEB_DIR=${TEMP_DIR}/deb
# <name>_<version>-<revision>_<architecture>.deb
DEB_FILE_NAME=${DIST}/${APPNAME}_${VERSION}-${REVISION}_${ARCH}.deb

LINTIAN=lintian --no-tag-display-limit
DPKG_DEB=dpkg-deb --debug
DEBSIGS=debsigs

all: clean build package

env:
	mkdir -p ${DIST}

clean:
	rm -rf ${DIST} ${TEMP_DIR}

lint-env:
	( which gosec &>/dev/zero && gosec --version | grep -qs $(GOSEC_VERSION) ) || go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
	( which golangci-lint &>/dev/zero && golangci-lint --version | grep -qs $(GOLANGCILINT_VERSION) ) || go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCILINT_VERSION)
	( which govulncheck &>/dev/zero ) || go install golang.org/x/vuln/cmd/govulncheck@$(VULNCHECK_VERSION)

lint: lint-env
	golangci-lint --timeout 10m run -v ./...
	gosec ./...
	govulncheck ./...

lint-fix: lint-env
	golangci-lint run -v --fix ./...

test: test-short
	go test ${VENDOR} ./...

test-short:
	go test ${VENDOR} -race -short

build: env
	CGO_ENABLED=0 go build -ldflags "-X 'github.com/halacs/haltonika/version.Version=?version?' -X 'github.com/halacs/haltonika/version.BuildDate=?date?'" -v -o ${DIST}/${APPNAME} .

.PHONY: package
package:
	${MAKE} package-debian

.ONESHELL: package-debian
package-debian: env build
	# Ensure source files of the debian package are there
	mkdir -p ${DEB_DIR}
	cp -r ${SRC_DIR_DEBIAN}/* ${DEB_DIR}/

	# Add binary files
	cp ${DIST}/${APPNAME} ${DEB_DIR}/usr/bin/haltonika
	chmod +x ${DEB_DIR}/usr/bin/haltonika

	mkdir -p ${DEB_DIR}debian && touch ${DEB_DIR}debian/control   # don't know why but this file is needed for dpkg-shlibdeps to work
	DEPENDS="$(find ${DEB_DIR} -executable -type f -exec dpkg-shlibdeps -O {} + | sed 's/shlibs:Depends=//g' )"
	INSTALLED_SIZE="$(du ${DEB_DIR} --exclude DEBIAN --summarize | cut -f1)"
	rm ${DEB_DIR}debian/control && rmdir ${DEB_DIR}debian

	# Build debian package file
	#mkdir ${DEB_DIR}/DEBIAN
	cat <<EOF  > ${DEB_DIR}/DEBIAN/control
	Package: ${APPNAME}
	Source: halacs.hu
	Version: ${VERSION}
	Architecture: ${ARCH}
	Maintainer: Gábor Nyíri (halacs.hu)
	Installed-Size: ${INSTALLED_SIZE}
	#Depends: ${DEPENDS}
	#Suggests: 
	#Breaks: 
	Replaces: ${APPNAME} (<< ${VERSION})
	Section: net
	Priority: optional
	Homepage: https://halacs.hu/
	Description: Microservice to collect data from Teltonika GPS devices.
	Original-Maintainer: Gábor Nyíri
	EOF

	# List configuration files
	find ${DEB_DIR}/etc -type f | sed 's!'${DEB_DIR}'!!g' > ${DEB_DIR}/DEBIAN/conffiles

	# Create MD5 hases for all files
	find ${DEB_DIR} -type f -exec md5sum {} + | grep -v '/DEBIAN/' | sed 's!'${DEB_DIR}/'!!g' > ${DEB_DIR}/DEBIAN/md5sums

	# Generate deb package
	${DPKG_DEB} --build --root-owner-group ${DEB_DIR} ${DEB_FILE_NAME}

	# Sign deb package
	#${DEBSIGS} --sign=origin -k FB4DCAD16D547D4EF5D0844E4AB1940A2044CCC4 ${DEB_FILE_NAME}
	#${DEBSIGS} --sign=maint -k FB4DCAD16D547D4EF5D0844E4AB1940A2044CCC4 ${DEB_FILE_NAME}
	#${DEBSIGS} --list ${DEB_FILE_NAME}

	# Linting generated deb file
	#${LINTIAN} ${DEB_FILE_NAME}

	echo "Done."
	ls -lah ${DEB_FILE_NAME}
	echo "${DEB_FILE_NAME}"


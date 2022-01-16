go_version = 1.17.6

all: version checksums.go

staticgotool/gotool.amd64.sqfs: Dockerfile
	docker build --build-arg=GO_VERSION=$(go_version) --rm -t gobuild .
	docker run --rm -v $$(pwd)/staticgotool:/tmp/bins gobuild cp -r /bins/ /tmp/

version:
	echo 'package gotool\n\nconst GoVersion = "$(go_version)"' > version.go

checksums.go: staticgotool/gotool.amd64.sqfs staticgotool/gotool.arm64.sqfs staticgotool/gotool.arm.sqfs
	( \
	echo 'package gotool\n\nvar checksums = map[string]string{'; \
	sha256sum staticgotool/*.sqfs | sed "s#^\([a-z0-9]*\)\ *staticgotool/gotool.\([^.]*\).sqfs#\t\"\2\": \"\1\",#g" | sort; \
	echo '}'; \
	) | gofmt > $@

clean:
	rm -f version.go checksums.go

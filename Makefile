go_version = 1.17.6

all: staticgotool/amd64/gotool.sqfs00 version

staticgotool/amd64/gotool.sqfs00: Dockerfile
	docker build --build-arg=GO_VERSION=$(go_version) --rm -t gobuild .
	docker run --rm -v $$(pwd)/staticgotool:/tmp/bins gobuild cp -r /bins/ /tmp/

version:
	echo 'package gotool\n\nconst GoVersion = "$(go_version)"' > version.go

clean:
	rm -f staticgotool/*/*.sqfs* version.go

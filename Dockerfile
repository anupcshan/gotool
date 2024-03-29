FROM golang:bullseye

RUN apt-get update && apt-get install -y curl squashfs-tools

ARG GO_VERSION
RUN mkdir /bins && curl -sL https://go.dev/dl/go$GO_VERSION.src.tar.gz -o /tmp/go.src.tar.gz

ENV SOURCE_DATE_EPOCH 1600000000

RUN mkdir /src && cd /src && tar xf /tmp/go.src.tar.gz && cd go/src && \
    CGO_ENABLED=0 GOARCH=amd64 ./make.bash && \
    mksquashfs ../ /bins/gotool.amd64.sqfs -noappend && rm -rf /src
RUN mkdir /src && cd /src && tar xf /tmp/go.src.tar.gz && cd go/src && \
    CGO_ENABLED=0 GOARCH=arm64 ./make.bash && rm -rf ../pkg/linux_amd64 ../pkg/tool/linux_amd64 && mv ../bin/linux_arm64/* ../bin && \
    mksquashfs ../ /bins/gotool.arm64.sqfs -noappend && rm -rf /src
RUN mkdir /src && cd /src && tar xf /tmp/go.src.tar.gz && cd go/src && \
    CGO_ENABLED=0 GOARCH=arm GOARM=7 ./make.bash && rm -rf ../pkg/linux_amd64 ../pkg/tool/linux_amd64 && mv ../bin/linux_arm/* ../bin && \
    mksquashfs ../ /bins/gotool.arm.sqfs -noappend && rm -rf /src

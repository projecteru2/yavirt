FROM projecteru2/footstone:yavirt-prebuild AS BUILD

# make binary
RUN git clone https://github.com/projecteru2/yavirt.git /go/src/github.com/projecteru2/yavirt
WORKDIR /go/src/github.com/projecteru2/yavirt
ARG KEEP_SYMBOL
RUN make deps && make && ./bin/yavirtd --version

FROM alpine:latest

RUN mkdir /etc/yavirt/
LABEL ERU=1
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/bin/yavirtd /usr/bin/yavirtd
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/bin/yavirtctl /usr/bin/yavirtctl
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/internal/virt/template/disk.xml /etc/yavirt/disk.xml
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/internal/virt/template/guest.xml /etc/yavirt/guest.xml

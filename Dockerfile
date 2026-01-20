FROM ubuntu:noble AS BUILD

# make binary
# RUN git clone https://github.com/projecteru2/yavirt.git /go/src/github.com/projecteru2/yavirt
COPY . /go/src/github.com/projecteru2/yavirt
WORKDIR /go/src/github.com/projecteru2/yavirt
ARG KEEP_SYMBOL
RUN apt update
RUN apt install -y golang-1.25 build-essential libvirt-dev make genisoimage libguestfs-dev libcephfs-dev librbd-dev librados-dev git
# RUN snap install go --classic
ENV PATH="$PATH:/usr/lib/go-1.25/bin/"

RUN go version
RUN make deps
RUN make && ./bin/yavirtd --version

FROM ubuntu:noble

RUN mkdir /etc/yavirt/ && \
    apt update && \
    apt install -y libvirt-dev libguestfs-dev genisoimage libcephfs-dev librbd-dev librados-dev

LABEL ERU=1
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/bin/yavirtd /usr/bin/yavirtd
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/bin/yavirtctl /usr/bin/yavirtctl
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/internal/virt/domain/templates/disk.xml /etc/yavirt/disk.xml
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/internal/virt/domain/templates/guest.xml /etc/yavirt/guest.xml

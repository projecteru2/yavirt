FROM ubuntu:jammy AS BUILD

# make binary
# RUN git clone https://github.com/projecteru2/yavirt.git /go/src/github.com/projecteru2/yavirt
COPY . /go/src/github.com/projecteru2/yavirt
WORKDIR /go/src/github.com/projecteru2/yavirt
ARG KEEP_SYMBOL
RUN sed -i 's@//.*archive.ubuntu.com@//mirrors.ustc.edu.cn@g' /etc/apt/sources.list
RUN apt update
RUN apt install -y golang-1.20 build-essential libvirt-dev make genisoimage libguestfs-dev libcephfs-dev librbd-dev librados-dev
RUN apt install -y git
# RUN snap install go --classic
ENV PATH="$PATH:/usr/lib/go-1.20/bin/"

RUN go version
RUN make deps CN=1
RUN make && ./bin/yavirtd --version

FROM ubuntu:jammy

RUN mkdir /etc/yavirt/ && \
    sed -i 's@//.*archive.ubuntu.com@//mirrors.ustc.edu.cn@g' /etc/apt/sources.list && \
    apt update && \
    apt install -y libvirt-dev libguestfs-dev genisoimage libcephfs-dev librbd-dev librados-dev

LABEL ERU=1
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/bin/yavirtd /usr/bin/yavirtd
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/bin/yavirtctl /usr/bin/yavirtctl
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/internal/virt/domain/templates/disk.xml /etc/yavirt/disk.xml
COPY --from=BUILD /go/src/github.com/projecteru2/yavirt/internal/virt/domain/templates/guest.xml /etc/yavirt/guest.xml

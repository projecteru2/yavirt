# Building a local dev. environment (Ubuntu)

## Dependencies

- build-essential
- qemu
- qemu-kvm
- libvirt-dev
- make

### Installing libext2fs

```bash
cd /tmp
curl -LOv http://prdownloads.sourceforge.net/e2fsprogs/e2fsprogs-1.47.3.tar.gz
tar -xzf e2fsprogs-1.47.3.tar.gz
cd e2fsprogs-1.47.3
./configure
make && make install && make install-libs
```

### Installing libguestfs (v1.58)

```bash
apt install -y gperf genisoimage flex bison libncurses-dev libpcre2-dev libaugeas-dev libmagic-dev libjansson-dev libhivex-dev libyara-dev supermin ocaml ocaml-findlib libhivex-ocaml-dev
./init-libguestfs.sh
apt install -y libguestfs-dev
```

# Running as a systemd service

## Runtime dependencies

- qemu
- qemu-kvm
- libvirt
- libguestfs

## Config file (.toml)

```ini
log_level = "info"
virt_dir = "/opt/yavirtd"
calico_pools = ["pool"]
etcd_prefix = "/yavirt/v1"
etcd_endpoints = ["http://127.0.0.1:2379"]
```

## Systemd unit file

```
[Unit]
Description=yavirtd
After=network.target
Wants=network-online.target

[Service]
User=root
PermissionsStartOnly=true
ExecStart=/usr/local/bin/yavirtd /etc/yavirt/yavirtd.toml
Restart=always
RestartSec=8s

[Install]
WantedBy=multi-user.target
```

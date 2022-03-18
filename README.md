# How to run as a systemd service

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


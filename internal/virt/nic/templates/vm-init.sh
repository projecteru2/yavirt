#!/bin/bash

# initialize NIC
# usage:
# vm-init.sh ip1 gw1 ip2 gw2
ifs=$(ip l | grep state | awk -F ': ' '{ if($2 != "lo" ) {print $2} }')

for ifname in $ifs
do
    if ip a show dev "$ifname" | grep -q 'inet '; then
        echo "The interface $ifname has an IP address."
        shift 2
        continue
    fi
    ip_addr=$1
    gw_addr=$2

    network="/etc/systemd/network/10-$ifname.network"
    cat << EOF > $network
[Match]
Name=$ifname

# [Network]
# Gateway=$gw_addr

[Address]
Address=$ip_addr

[Route]
Gateway=$gw_addr
# Destination=10.0.0.0/8
GatewayOnlink=yes
EOF
    chmod 644 "/etc/systemd/network/10-$ifname.network"
    # ip r add default via 169.254.1.1 dev $ifname onlink
    shift 2
done

systemctl restart systemd-networkd

# prepare dns if neccessary
dnsOutput=$(dig +short baidu.com)
if [ -z "$dnsOutput" ]
then
    echo "Setting DNS..."
    mkdir /etc/systemd/resolved.conf.d/
    cat << EOF > /etc/systemd/resolved.conf.d/dns_servers.conf
[Resolve]
DNS=8.8.8.8 1.1.1.1
EOF

    systemctl restart systemd-resolved
fi
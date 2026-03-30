#!/bin/bash

OUTPUT_DIR="./scheduler/configs"
rm -f ${OUTPUT_DIR}/*.config

declare -A DEVICE_TYPES=(
    ["AR"]="Aggregation Router"
    ["SR"]="Service Router"
    ["CR"]="Core Router"
    ["SW"]="Switch"
    ["CSW"]="Client Switch"
    ["IAS"]="Intermediate Aggregation Switch"
    ["VGW"]="Voice Gateway"
    ["TRS"]="ToR Switch"
)

TERRITORIES=("ekb" "ntg" "kur")

generate_router_config() {
    local name=$1
    local type=$2
    local territory=$3
    local num=$4
    
    cat > "${OUTPUT_DIR}/${name}.config" << EOF
! ${type} Configuration
! Device: ${name}
! Type: ${DEVICE_TYPES[$type]}
! Territory: ${territory}
! Generated for load testing
! Date: $(date +%Y-%m-%d\ %H:%M:%S)

version 15.9
service timestamps debug datetime msec
service timestamps log datetime msec
service password-encryption
!
hostname ${name}
!
boot-start-marker
boot-end-marker
!
no aaa new-model
clock timezone GMT 5 0
!
EOF

    # NTP
    cat >> "${OUTPUT_DIR}/${name}.config" << 'EOF'
ntp server 10.100.1.1
ntp server 10.100.1.2
ntp server 10.100.1.3 prefer
!
EOF

    # Domain and Users
    cat >> "${OUTPUT_DIR}/${name}.config" << EOF
ip domain-name ${territory}.local
ip name-server 10.0.0.53
ip name-server 10.0.0.54
!
username admin privilege 15 secret 5 \$1\$ABCD\$EFGHijKlmNopQrStUvWxYz
username operator privilege 5 secret 5 \$1\$WXYZ\$ijKlAbCdEfGhIjKlMnOp
!
EOF

    # Interfaces - больше для router типов
    for port in $(seq 0 49); do
        slot=$((port / 10))
        port_in_slot=$((port % 10))
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
interface GigabitEthernet${slot}/${port_in_slot}
 description Port-${port}-${name}-Uplink
 ip address 10.${num}.${port}.1 255.255.255.0
 ip nat inside
 ip virtual-reassembly in
 load-interval 300
 carrier-delay msec 100
 no shutdown
!
EOF
    done

    # Loopback
    for lo in $(seq 0 9); do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
interface Loopback${lo}
 description Loopback-${lo}-Management-${name}
 ip address 10.255.255.${lo} 255.255.255.255
 ip ospf network point-to-point
!
EOF
    done

    # VLANs
    for vlan in $(seq 1 20); do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
vlan ${vlan}
 name VLAN-${vlan}-${name}
!
EOF
    done

    # OSPF
    cat >> "${OUTPUT_DIR}/${name}.config" << 'EOF'
router ospf 100
 router-id 10.255.255.1
 auto-cost reference-bandwidth 100000
 passive-interface default
 no passive-interface GigabitEthernet0/0
 no passive-interface GigabitEthernet0/1
 network 10.0.0.0 0.255.255.255 area 0
!
ipv6 router ospf 100
 router-id 10.255.255.1
 passive-interface default
!
EOF

    # BGP
    cat >> "${OUTPUT_DIR}/${name}.config" << 'EOF'
router bgp 65000
 bgp router-id 10.255.255.1
 bgp log-neighbor-changes
 no bgp default ipv4-unicast
!
 address-family ipv4 unicast
EOF
    for peer in $(seq 1 10); do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
  neighbor 10.200.${peer}.1 remote-as ${peer}0001
  neighbor 10.200.${peer}.1 description Peer-${peer}
  neighbor 10.200.${peer}.1 activate
EOF
    done
    cat >> "${OUTPUT_DIR}/${name}.config" << 'EOF'
 exit-address-family
!
EOF

    # ACLs
    for acl_num in 100 101 102 103 104; do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
ip access-list extended ACL-${acl_num}
EOF
        for rule in $(seq 10 5 500); do
            cat >> "${OUTPUT_DIR}/${name}.config" << EOF
 permit tcp any host 192.168.${acl_num}.${rule} eq ${rule}
 permit udp any host 192.168.${acl_num}.${rule} eq ${rule}
 permit icmp any host 192.168.${acl_num}.${rule}
 deny ip any any log
EOF
        done
        cat >> "${OUTPUT_DIR}/${name}.config" << 'EOF'
!
EOF
    done

    # Standard ACLs
    for acl_num in 1 2 3 4 5; do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
ip access-list standard ACL-STD-${acl_num}
 permit 10.${acl_num}.0.0 0.0.255.255
 permit 172.16.${acl_num}.0 0.0.0.255
 permit 192.168.${acl_num}.0 0.0.0.255
 deny any log
!
EOF
    done

    # Route Maps
    for rm in $(seq 1 5); do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
route-map RM-${rm} permit 10
 match ip address prefix-list PL-${rm}
 set local-preference ${rm}00
 set community ${rm}:1
!
route-map RM-${rm} permit 20
 set local-preference ${rm}50
!
EOF
    done

    # Prefix Lists
    for pl in $(seq 1 10); do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
ip prefix-list PL-${pl} seq 10 permit 10.${pl}.0.0/16 le 24
ip prefix-list PL-${pl} seq 20 permit 172.${pl}.0.0/16 le 24
ip prefix-list PL-${pl} seq 30 permit 192.168.${pl}.0/24
!
EOF
    done

    # QoS
    for policy in $(seq 1 5); do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
policy-map PM-${policy}
 class CLASS-${policy}
  priority percent ${policy}0
  shape average ${policy}000000
!
 class class-default
  fair-queue
!
class-map match-any CLASS-${policy}
 match access-group name ACL-100
!
EOF
    done

    # NAT
    cat >> "${OUTPUT_DIR}/${name}.config" << 'EOF'
ip nat pool NAT-POOL 10.255.0.1 10.255.0.254 netmask 255.255.255.0
ip nat inside source list 1 pool NAT-POOL overload
ip nat inside source static tcp 10.0.0.1 22 203.0.113.1 2222
ip nat inside source static tcp 10.0.0.1 80 203.0.113.1 8080
!
ip nat translation timeout 3600
ip nat translation tcp-timeout 3600
!
EOF

    # SNMP
    cat >> "${OUTPUT_DIR}/${name}.config" << EOF
snmp-server community public RO
snmp-server community private RW
snmp-server location ${territory^^}-DataCenter-${name}
snmp-server contact noc@${territory}.local
snmp-server enable traps snmp authentication linkdown linkup coldstart warmstart
snmp-server host 10.100.100.1 version 2c public
snmp-server host 10.100.100.2 version 3 priv public
!
EOF

    # Syslog
    cat >> "${OUTPUT_DIR}/${name}.config" << 'EOF'
logging host 10.100.200.1
logging host 10.100.200.2
logging trap informational
logging source-interface Loopback0
logging buffered 65536 informational
!
EOF

    # Additional blocks for size
    for block in $(seq 1 800); do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF

! Block ${block} - ${name} - ${type} Device on ${territory}
! Extended configuration block for load testing purposes
interface GigabitEthernet3/${block}
 description Extended-Port-${block}-${name}-Extended-Description-Text-Additional-Info
 ip address 172.16.${block}.1 255.255.255.0
 ip nat outside
 standby ${block} ip 172.16.${block}.254
 standby ${block} priority 110
 load-interval 300
 carrier-delay msec 100
 no shutdown
!
interface GigabitEthernet3/$((block+800))
 description Backup-Port-${block}-${name}-Extended-Description
 ip address 172.16.$((block+800)).1 255.255.255.0
 standby ${block} ip 172.16.${block}.254
 standby ${block} priority 100
 no shutdown
!
router bgp 65000
 address-family ipv4 unicast
  neighbor 10.200.${block}.254 remote-as $((block*1000))
  neighbor 10.200.${block}.254 description Extended-Peer-${block}
  neighbor 10.200.${block}.254 update-source Loopback0
!
ip access-list extended ACL-EXT-${block}
 permit tcp any host 10.${block}.0.1 eq 80
 permit tcp any host 10.${block}.0.1 eq 443
 permit udp any host 10.${block}.0.1 eq 53
 deny ip any any log
!
route-map RM-${block} permit ${block}
 match ip address prefix-list PL-${block}
 set metric ${block}000
!
EOF
    done

    # Serial/Tunnel interfaces
    for ext in $(seq 1 400); do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
interface Serial${ext}/0
 description Serial-Link-${ext}-${name}
 ip address 10.${num}.${ext}.1 255.255.255.252
 encapsulation ppp
 clock rate 64000
 no shutdown
!
interface Tunnel${ext}
 description Tunnel-${ext}-VPN-${name}
 ip address 10.255.${ext}.1 255.255.255.252
 tunnel source Loopback0
 tunnel destination 10.255.255.${ext}
!
EOF
    done

    # More ACLs
    for acl_ext in $(seq 200 249); do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
ip access-list extended ACL-${acl_ext}
 permit tcp any any eq ${acl_ext}
 permit udp any any eq ${acl_ext}
 permit icmp any any echo
 deny ip any any log
!
EOF
    done

    # More route maps
    for rm_ext in $(seq 10 39); do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
route-map RM-${rm_ext} permit 10
 match ip address ACL-${rm_ext}
 set metric-type type-1
 set local-preference ${rm_ext}0
!
route-map RM-${rm_ext} permit 20
 set metric-type type-2
!
EOF
    done

    # More prefix lists
    for pl_ext in $(seq 20 59); do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
ip prefix-list PL-${pl_ext} seq 5 permit 10.${pl_ext}.0.0/16 le 28
ip prefix-list PL-${pl_ext} seq 10 permit 172.${pl_ext}.0.0/12 le 24
ip prefix-list PL-${pl_ext} seq 15 permit 192.168.${pl_ext}.0/24
!
EOF
    done

    # MAC and ARP
    for mac in $(seq 1 400); do
        cat >> "${OUTPUT_DIR}/${name}.config" << EOF
mac address-table static 0000.${mac:0:2}.${mac:2:2} vlan $((mac % 20 + 1)) interface GigabitEthernet0/$((mac % 5))
arp 10.${num}.${mac}.1 0000.${mac:0:2}.${mac:2:2} ARPA
!
EOF
    done

    # Final section
    cat >> "${OUTPUT_DIR}/${name}.config}" << 'EOF'

! Multicast
ip multicast-routing distributed
ip pim rp-address 10.255.255.1
!
! Control Plane Policing
control-plane
 service-policy input COP-POLICY-IN
!
class-map match-any COP-CLASS-IN
 match access-group name ACL-100
 match protocol bgp
!
policy-map COP-POLICY-IN
 class COP-CLASS-IN
  police rate 5000000 burst 5000000 conform-action transmit exceed-action drop
!
! NetFlow
flow exporter EXPORTER-1
 destination 10.100.100.10
 source Loopback0
 transport udp 9995
!
flow monitor MONITOR-1
 exporter EXPORTER-1
 record netflow original-input
!
! EEM
event manager applet SECURITY-MONITOR
 event syslog pattern "AUTHENTICATION FAILURE"
 action 1.0 syslog msg "Security alert on device"
!
event manager applet INTERFACE-DOWN
 event interface name GigabitEthernet0/0 parameter errdisable
 action 1.0 syslog msg "Interface error disabled"
!
end
EOF

    echo "Created: ${name}.config"
}

idx=1
for type in AR SR CR SW CSW IAS VGW TRS; do
    for t in "${TERRITORIES[@]}"; do
        if [ $idx -gt 20 ]; then
            break 2
        fi
        num=$(printf "%03d" $idx)
        name="${type}${num}-${t}"
        generate_router_config "$name" "$type" "$t" "$idx"
        idx=$((idx + 1))
    done
done

echo ""
echo "=== Summary ==="
echo "Total files: $(ls -1 ${OUTPUT_DIR}/*.config 2>/dev/null | wc -l)"
echo "Total size: $(du -sh ${OUTPUT_DIR} 2>/dev/null | cut -f1)"
echo ""
echo "Device types:"
ls ${OUTPUT_DIR}/*.config 2>/dev/null | xargs -n1 basename | sed 's/\.config$//' | sort

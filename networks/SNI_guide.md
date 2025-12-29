# setting up nginx SNI proxy on port 8443:

#### Add this at the end of your nginx.conf file, after the http block

```bash
stream {
    resolver 1.1.1.1 8.8.8.8 valid=300s;

    log_format stream_log '$remote_addr [$time_local] '
                          '$ssl_preread_server_name '
                          '$status $bytes_sent $bytes_received $session_time';

    access_log /var/log/nginx/sni-proxy-access.log stream_log;
    error_log  /var/log/nginx/sni-proxy-error.log;

    map $ssl_preread_server_name $backend {
        api.openweathermap.org api_openweather;
        shahraranews.ir        shahrara;
        default                reject;
    }

    upstream api_openweather {
        server api.openweathermap.org:443;
    }

    upstream shahrara {
        server shahraranews.ir:443;
    }

    server {
        listen 8443;
        ssl_preread on;
        proxy_pass $backend;
        proxy_connect_timeout 10s;
        proxy_timeout 300s;
    }
}
```

# Script to update a docker network based on the SNI and URL settings:

```bash
#!/bin/bash
# File: sni_proxy_safe.sh
# Purpose: Safely add SNI proxy for api.openweathermap.org in Docker containers
#          and apply iptables DNAT only for that fake IP.

# Configurable variables
SNI_PROXY_IP="5.252.216.164"   # Your SNI proxy server
FAKE_IP="10.255.255.2"         # Fake IP to map api.openweathermap.org
TARGET_HOST="shahraranews.ir"
PORT="443"

echo "[*] Setting up safe SNI proxy for Docker containers..."

# 1. Update /etc/hosts in all running containers
echo "[*] Updating /etc/hosts in running containers..."
CONTAINERS=$(docker ps -q)
for CONTAINER in $CONTAINERS; do
    # Avoid duplicate entries
    docker exec -i $CONTAINER sh -c "grep -q '$TARGET_HOST' /etc/hosts || echo '$FAKE_IP $TARGET_HOST' >> /etc/hosts"
    echo " - Updated container $CONTAINER"
done

# 2. Apply iptables DNAT only for traffic destined to the fake IP
echo "[*] Applying iptables DNAT rules..."

# Get all Docker networks with subnets
NETWORKS=$(docker network ls -q)
for NET_ID in $NETWORKS; do
    SUBNETS=$(docker network inspect $NET_ID --format '{{range .IPAM.Config}}{{.Subnet}}{{end}}')
    for SUBNET in $SUBNETS; do
        if [ -n "$SUBNET" ]; then
            echo " - Adding DNAT for subnet $SUBNET destined to $FAKE_IP"
            # Avoid duplicate rules
            iptables -t nat -C PREROUTING -s $SUBNET -d $FAKE_IP -p tcp --dport $PORT -j DNAT --to-destination $SNI_PROXY_IP:8443 2>/dev/null || \
            iptables -t nat -A PREROUTING -s $SUBNET -d $FAKE_IP -p tcp --dport $PORT -j DNAT --to-destination $SNI_PROXY_IP:8443
        fi
    done
done

echo "[*] Safe SNI proxy setup complete. Only traffic for $TARGET_HOST is redirected."
```

# script to run the SNI script on each new docker container start 

```bash
#!/bin/bash

# Infinite loop to listen for Docker events
docker events --filter event=start | while read event; do
    /root/scripts/shahrara
    /root/scripts/weather
done
```


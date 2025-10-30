Below is an in‐depth, step-by-step guide that explains what a GRE tunnel is and how it can be set up and tested, followed by a tutorial for dnsmasq with a real‑world example. Each section is designed to be simple and easy to understand.

---

## Part 1. GRE Tunnel

### What Is a GRE Tunnel?

GRE stands for Generic Routing Encapsulation. It is a tunneling protocol developed by Cisco that allows you to encapsulate a wide variety of network layer protocols inside virtual point-to-point links over an IP network. In simple terms, a GRE tunnel creates a “pipe” between two endpoints so that data can travel securely (or simply in an encapsulated format) from one point to another even over intermediate networks.

### Why Use a GRE Tunnel?

- **Encapsulation of Traffic:** You can encapsulate different protocols that might not normally traverse an IP network.
- **Traversing Untrusted Networks:** It helps in encapsulating the original packet so that even if the route has untrusted or incompatible segments, your data is delivered.
- **Connecting Disparate Networks:** It is often used in virtual private networks (VPNs) or to connect remote offices securely.

### How to Set Up and Test a GRE Tunnel: A Step-by-Step Tutorial

We’ll use two Linux hosts (or virtual machines) as our tunnel endpoints. Let’s assume:
- Host A has IP address 192.168.1.10 (public IP or reachable over the network)
- Host B has IP address 192.168.2.10

#### 1. Prerequisites

- Both hosts should be running a Linux distribution (like Ubuntu, CentOS, etc.).
- You need root or appropriate sudo privileges.
- Ensure that the GRE module is loaded on both systems (usually it is).

#### 2. Enabling GRE Module (if needed)

On both hosts, you might need to load the GRE kernel module. Run:

```bash
sudo modprobe ip_gre
```

To ensure it loads on boot, add it to `/etc/modules` (file location may vary by distribution).

#### 3. Create the GRE Tunnel on Host A

On Host A, execute the following commands:

```bash
# Create the GRE tunnel interface (named gre1)
sudo ip tunnel add gre1 mode gre remote 192.168.2.10 local 192.168.1.10 ttl 255

# Set an IP address for the tunnel interface
sudo ip addr add 10.0.0.1/30 dev gre1

# Bring the tunnel interface up
sudo ip link set gre1 up
```

Explanation:
- We’re creating a tunnel interface named `gre1` with Host A’s IP as local and Host B’s IP as remote.
- The tunnel interface gets an IP address from a separate network (10.0.0.0/30) which is used for communication between the tunnel endpoints.
- Bringing the interface up means it can now pass traffic.

#### 4. Create the GRE Tunnel on Host B

On Host B, run similar commands but swap the local and remote parameters:

```bash
# Create the GRE tunnel interface (named gre1)
sudo ip tunnel add gre1 mode gre remote 192.168.1.10 local 192.168.2.10 ttl 255

# Set an IP address for the tunnel interface
sudo ip addr add 10.0.0.2/30 dev gre1

# Bring the tunnel interface up
sudo ip link set gre1 up
```

#### 5. Test the GRE Tunnel

Now that both tunnel interfaces are configured, test connectivity:

- On Host A, ping the GRE interface on Host B:

  ```bash
  ping -c 4 10.0.0.2
  ```

- On Host B, ping the GRE interface on Host A:

  ```bash
  ping -c 4 10.0.0.1
  ```

If the pings are successful, the GRE tunnel is working. The data is encapsulated and passed between the two hosts through the underlying (public or intermediate) networks.

### Troubleshooting Tips

- Ensure there are no firewalls blocking GRE (protocol 47) between the hosts.
- Verify that both hosts have matching configurations (correct local/remote IP addresses).
- Use tools like `tcpdump` on the physical interface to see the GRE packets.

---

## Part 2. dnsmasq

### What Is dnsmasq?

dnsmasq is a lightweight, easy-to-configure DNS forwarder and DHCP server commonly used in small networks. It can provide:
- Local DNS caching: speeding up DNS queries.
- Local resolving of hostnames.
- DHCP services to hand out IP addresses and options.
- TFTP services for network booting etc.

### A Real‑World Example: Using dnsmasq as a DHCP and DNS Server

Imagine you have a small home or office network and want a single server to provide IP addresses via DHCP and resolve local hostnames, as well as forward DNS queries for external domains.

### Step-by-Step Tutorial for dnsmasq

#### 1. Installation

Install dnsmasq using your package manager.

For Ubuntu/Debian:

```bash
sudo apt update
sudo apt install dnsmasq
```

For CentOS/RHEL:

```bash
sudo yum install dnsmasq
```

#### 2. Configure dnsmasq

The main configuration file is usually `/etc/dnsmasq.conf`. Make a backup before editing:

```bash
sudo cp /etc/dnsmasq.conf /etc/dnsmasq.conf.backup
```

Open the configuration file in your editor:

```bash
sudo nano /etc/dnsmasq.conf
```

Add or modify settings to enable DHCP and assign a local DNS domain. For example:

```conf
# Specify the interface dnsmasq should listen on; replace 'eth0' with your network interface.
interface=eth0

# Specify the local DNS domain
domain=localnet

# Set the DHCP range. For example, handing out addresses from 192.168.1.100 to 192.168.1.150 with a 12-hour lease.
dhcp-range=192.168.1.100,192.168.1.150,12h

# Optionally, specify a router (gateway) to hand out.
dhcp-option=3,192.168.1.1

# Specify a local DNS entry permanently (hostname and IP mapping)
address=/myprinter.localnet/192.168.1.200

# Enable logging for troubleshooting
log-queries
log-dhcp
```

Save the file and exit your editor.

#### 3. Restart dnsmasq

For the changes to take effect, restart the dnsmasq service:
└─bareos--vg-bareos--lv 252:0    0  100G  0 lvm  /mnt/bareos
On systemd-based systems:

```bash
sudo systemctl restart dnsmasq
```

Check the status:

```bash
sudo systemctl status dnsmasq
```

#### 4. Testing dnsmasq

- **Testing DNS resolution:**
  
  From a client on the network (or from the server itself), run:
  
  ```bash
  nslookup myprinter.localnet
  ```
  
  It should resolve to 192.168.1.200.
  
  Also check an external query:
  
  ```bash
  nslookup google.com
  ```

- **Testing DHCP functionality:**

  - Disconnect a device from the network and reconnect (or manually force a DHCP renewal).
  - Check that the device receives an IP address in the range you specified (192.168.1.100 to 192.168.1.150).
  - Inspect the DHCP lease file (commonly `/var/lib/misc/dnsmasq.leases` or `/var/lib/dnsmasq/dnsmasq.leases`) to see which IPs have been assigned.

- **Checking logs:**

  If you enabled logging in the configuration, check syslog or the dnsmasq-specific log file (location may vary by distribution) to view queries and lease assignments.

### Real‑World Use Case Scenario

Imagine you run a small office with a single router, a couple of network printers, and several computers. Here’s how dnsmasq helps:
- Any device that connects to the network gets assigned an IP address automatically.
- When employees type names like “myprinter.localnet” in their browsers or printer setup utilities, dnsmasq instantly resolves it to the correct IP.
- dnsmasq caches frequent external DNS queries (like for common websites), speeding up browsing.
- It all happens in one small, simple daemon instead of needing a full-blown enterprise DHCP/DNS server.

---

## Conclusion

In summary:

- A GRE tunnel encapsulates packets to allow them to traverse networks—even allowing non-IP protocols to be carried over IP networks. The tutorial above shows how to set up a simple GRE tunnel between two Linux hosts.
- dnsmasq is a lightweight solution that can serve as both a DNS forwarder and a DHCP server. The dnsmasq tutorial provided demonstrates setting up a local, real-world network service, including configuring DHCP ranges and custom DNS mappings.

Both tools are powerful for network configuration and troubleshooting in different scenarios. With these tutorials, you should be well‑equipped to set up, test, and use both GRE tunnels and dnsmasq in your network environments.



Once the GRE tunnel is up and running (i.e., both endpoints can ping each other using their tunnel interface IPs), you can use it just like any other network link. Here are a few common ways to “use” a GRE tunnel in practical scenarios:

1. ️Routing Between Networks  
    • Imagine you have two separate subnets (e.g., 192.168.10.0/24 on Host A’s side and 192.168.20.0/24 on Host B’s side). You can use the GRE tunnel as a dedicated path between these subnets.  
    • Configure routing on both sides so that packets destined for the remote subnet are sent through the GRE tunnel interface.  
    • For example, on Host A you could add a route for 192.168.20.0/24 via the GRE tunnel IP (10.0.0.2), and similarly on Host B add a route for 192.168.10.0/24 via 10.0.0.1.  
    • As a result, a computer in the 192.168.10.0/24 network can communicate with a machine in the 192.168.20.0/24 network, with the GRE tunnel encapsulating the traffic between the two gateways.

2. ️Creating a Secure or Isolated Link  
    • GRE tunnels can be combined with additional security layers such as IPsec to provide encrypted connectivity over untrusted networks (such as the Internet).  
    • Even without encryption, you might choose GRE to “logically” separate traffic types or to simulate a private connection over a shared network.

3. ️Connecting Remote Offices or Virtual Data Centers  
    • Organizations often find that GRE tunnels are an easy way to connect different geographical locations.  
    • Traffic destined for one branch office or data center can be sent over an encrypted GRE tunnel from the headquarters to the remote location.  
    • The GRE tunnel makes the two physically separated networks behave as if they were directly connected.

4. ️Tunneling Protocols Not Normally Allowed  
    • GRE isn’t limited to just IP traffic. It encapsulates many network protocols. If you have an application that uses a non-IP protocol, you could encapsulate it in a GRE tunnel so that it’s transported over an IP network.  
    • For instance, routing protocols or even multicast traffic can be tunneled if configured correctly.

### Configuring Routing to Use the Tunnel

Once your GRE tunnel is active, here’s how you might set up static routes to send specific traffic through the tunnel:

- On Host A (gateway for Network A):

  For example, if your remote network (behind Host B) is 192.168.20.0/24, you add a route:

  ```
  sudo ip route add 192.168.20.0/24 via 10.0.0.2 dev gre1
  ```

- On Host B (gateway for Network B):

  Similarly, if the local network behind Host A is 192.168.10.0/24, you add the route:

  ```
  sudo ip route add 192.168.10.0/24 via 10.0.0.1 dev gre1
  ```

After adding these routes, any packet from the local network destined for the remote subnet will pass into the GRE tunnel. The tunnel encapsulates these packets and sends them to the other side, where they are decapsulated and forwarded to the destination.

### Testing GRE Tunnel Usage in a Routing Scenario

1.  From a host in Network A (say 192.168.10.5) set its default gateway (or a specific route) to point to Host A’s local GRE endpoint.  
2.  Likewise, a host in Network B (say 192.168.20.5) uses Host B as its gateway.  
3.  Now try to ping or access resources across networks. For example, from 192.168.10.5, ping 192.168.20.5.  
4.  If the routing is set up correctly and the GRE tunnel is functioning, you will see the traffic traveling across the tunnel.

### Summary

- With the GRE tunnel established (hosts respond to ping on the tunnel IPs), you convert it into a “virtual link” between the two endpoints.  
- You can then establish routes or even run routing protocols (like OSPF) across the tunnel, enabling different networks or remote offices to communicate as if they were directly connected.  
- This setup is useful for securely connecting branch offices, linking disparate networks, or even encapsulating non-IP protocols over an IP backbone.

Using the GRE tunnel is essentially about leveraging it in your overall network design by updating routing decisions to treat it as your link of choice for specific traffic.


---

# LVM:
question: i have ubuntu server with a raw disk called sdb with 100GB space i want to setup a partition and use LVM and name it bareos and mount it on /mnt/bareos 
step by step guide to set it up so that later i can easily extend it?

Here's a **step-by-step guide** to set up a raw disk (`/dev/sdb`) with **LVM** on an Ubuntu server. You'll create a partition, configure LVM, and mount it to `/mnt/bareos`, in a way that makes it **easy to extend later**.

---

### 🧰 Prerequisites

* You are **root** or have `sudo` privileges.
* Disk `/dev/sdb` is **unused** (no partitions or important data).

---

## ✅ Step 1: Create a Partition on /dev/sdb

```bash
sudo fdisk /dev/sdb
```

**In `fdisk`:**

* Press `n` to create a new partition.
* Select `p` for primary.
* Accept defaults for partition number, first sector, and last sector (use full disk).
* Press `t`, then type `8e` (LVM partition type).
* Press `w` to write and exit.

Now you have `/dev/sdb1` as an LVM partition.

---

## ✅ Step 2: Set Up LVM

### 2.1 Install LVM if not installed

```bash
sudo apt update
sudo apt install -y lvm2
```

### 2.2 Create Physical Volume

```bash
sudo pvcreate /dev/sdb1
```

### 2.3 Create Volume Group

```bash
sudo vgcreate bareos-vg /dev/sdb1
```

### 2.4 Create Logical Volume (use all space or partial)

Let’s say you want to use all space now:

```bash
sudo lvcreate -l 100%FREE -n bareos-lv bareos-vg
```

Or, for example, use 50GB for now:

```bash
sudo lvcreate -L 50G -n bareos-lv bareos-vg
```

---

## ✅ Step 3: Format and Mount the Logical Volume

### 3.1 Format with ext4 (or your preferred FS)

```bash
sudo mkfs.ext4 /dev/bareos-vg/bareos-lv
```

### 3.2 Create mount point

```bash
sudo mkdir -p /mnt/bareos
```

### 3.3 Mount it

```bash
sudo mount /dev/bareos-vg/bareos-lv /mnt/bareos
```

### 3.4 Add to `/etc/fstab` for persistence

```bash
echo '/dev/bareos-vg/bareos-lv /mnt/bareos ext4 defaults 0 2' | sudo tee -a /etc/fstab
```

---

## ✅ Step 4: Verify

```bash
df -h /mnt/bareos
sudo vgs
sudo lvs
```

---

## 📦 Step 5: Extend Later (When Needed)

When `/mnt/bareos` is full and you want to extend:

### 5.1 Add new disk or free space to the VG:

Example with a new disk `/dev/sdc`:

```bash
sudo fdisk /dev/sdc
# repeat fdisk steps → create partition → type 8e
sudo pvcreate /dev/sdc1
sudo vgextend bareos-vg /dev/sdc1
```

### 5.2 Extend the Logical Volume

Example: extend by 20GB

```bash
sudo lvextend -L +20G /dev/bareos-vg/bareos-lv
```

Or use all available space:

```bash
sudo lvextend -l +100%FREE /dev/bareos-vg/bareos-lv
```

### 5.3 Resize the Filesystem

```bash
sudo resize2fs /dev/bareos-vg/bareos-lv
```

---

## ✅ Done!

You're now using `/mnt/bareos` backed by LVM, ready for easy expansion as needed.




---


# GRE over IPSec:

Implementing GRE over IPSec on Ubuntu Server is a common and powerful way to create a secure site-to-site VPN. We'll use `strongSwan` for the IPSec part and the built-in kernel `gre` module for the tunnel.

Here is a complete, step-by-step guide.

### **Scenario**

We will connect two office networks over the internet.

*   **Site A (Server A):**
    *   Public IP (WAN): `203.0.113.10`
    *   Private LAN: `192.168.10.0/24`
    *   GRE Tunnel IP: `10.0.0.1`

*   **Site B (Server B):**
    *   Public IP (WAN): `198.51.100.20`
    *   Private LAN: `192.168.20.0/24`
    *   GRE Tunnel IP: `10.0.0.2`

*   **Pre-Shared Key (PSK):** `MySecretVPNKey` (Use a strong, random key in production!)

---

### **Phase 1: Prerequisites on Both Servers**

Run these commands on **both Server A and Server B**.

1.  **Update System and Install strongSwan:**
    ```bash
    sudo apt update
    sudo apt install strongswan -y
    ```

2.  **Enable IP Forwarding:**
    This allows the server to act as a router, forwarding packets between its interfaces.
    ```bash
    sudo sysctl -w net.ipv4.ip_forward=1
    ```
    To make this permanent, edit the sysctl configuration file:
    ```bash
    sudo nano /etc/sysctl.conf
    ```
    Uncomment or add the following line:
    ```
    net.ipv4.ip_forward=1
    ```
    Save the file (`Ctrl+X`, then `Y`, then `Enter`).

3.  **Configure Firewall (UFW):**
    IPSec uses specific ports and protocols. We need to allow them.
    ```bash
    # Allow SSH (so you don't lock yourself out)
    sudo ufw allow OpenSSH

    # Allow IPSec traffic
    sudo ufw allow 500/udp
    sudo ufw allow 4500/udp
    
    # The ESP protocol is crucial for IPSec
    sudo ufw allow ESP
    
    # Enable the firewall
    sudo ufw enable
    ```

---

### **Phase 2: Configuration on Server A**

Edit the configuration files on **Server A (203.0.113.10)**.

1.  **Configure IPSec (`/etc/ipsec.conf`):**
    ```bash
    sudo nano /etc/ipsec.conf
    ```
    Clear the file and add the following content. This sets up a "transport mode" tunnel, which is what we use for GRE over IPSec.

    ```ini
    config setup
        charondebug="all" # Use for debugging, set to "none" in production
        uniqueids=yes

    conn %default
        keyexchange=ikev2
        authby=secret
        type=transport # IMPORTANT: Transport mode encrypts only the payload
        left=%any
        leftid=203.0.113.10
        leftprotoport=17/0 # UDP protocol for GRE
        right=%any
        rightid=198.51.100.20
        rightprotoport=17/0 # UDP protocol for GRE
        ike=aes256-sha1-modp2048!
        esp=aes256-sha1!
        auto=add # Load this connection on startup

    conn site-a-to-site-b
        left=203.0.113.10
        right=198.51.100.20
    ```

2.  **Set the Pre-Shared Key (`/etc/ipsec.secrets`):**
    ```bash
    sudo nano /etc/ipsec.secrets
    ```
    Add this line, replacing the IPs and key with your own:
    ```
    203.0.113.10 198.51.100.20 : PSK "MySecretVPNKey"
    ```

3.  **Create the GRE Tunnel:**
    ```bash
    # Create the tunnel interface
    sudo ip tunnel add gre0 mode gre remote 198.51.100.20 local 203.0.113.10 ttl 255

    # Bring the interface up
    sudo ip link set gre0 up

    # Assign the tunnel IP address
    sudo ip addr add 10.0.0.1/30 dev gre0
    ```

4.  **Start and Enable strongSwan:**
    ```bash
    sudo systemctl restart strongswan-starter
    sudo systemctl enable strongswan-starter
    ```

5.  **Bring the IPSec Tunnel Up:**
    ```bash
    sudo ipsec up site-a-to-site-b
    ```

---

### **Phase 3: Configuration on Server B**

Now, do the mirror configuration on **Server B (198.51.100.20)**.

1.  **Configure IPSec (`/etc/ipsec.conf`):**
    ```bash
    sudo nano /etc/ipsec.conf
    ```
    Add the following content. Notice that `left` and `right` are swapped.
    ```ini
    config setup
        charondebug="all" # Use for debugging, set to "none" in production
        uniqueids=yes

    conn %default
        keyexchange=ikev2
        authby=secret
        type=transport # IMPORTANT: Transport mode encrypts only the payload
        left=%any
        leftid=198.51.100.20
        leftprotoport=17/0
        right=%any
        rightid=203.0.113.10
        rightprotoport=17/0
        ike=aes256-sha1-modp2048!
        esp=aes256-sha1!
        auto=add

    conn site-a-to-site-b
        left=198.51.100.20
        right=203.0.113.10
    ```

2.  **Set the Pre-Shared Key (`/etc/ipsec.secrets`):**
    ```bash
    sudo nano /etc/ipsec.secrets
    ```
    Add the line (the key must be identical):
    ```
    198.51.100.20 203.0.113.10 : PSK "MySecretVPNKey"
    ```

3.  **Create the GRE Tunnel:**
    ```bash
    # Create the tunnel interface
    sudo ip tunnel add gre0 mode gre remote 203.0.113.10 local 198.51.100.20 ttl 255

    # Bring the interface up
    sudo ip link set gre0 up

    # Assign the tunnel IP address
    sudo ip addr add 10.0.0.2/30 dev gre0
    ```

4.  **Start and Enable strongSwan:**
    ```bash
    sudo systemctl restart strongswan-starter
    sudo systemctl enable strongswan-starter
    ```

5.  **Bring the IPSec Tunnel Up:**
    ```bash
    sudo ipsec up site-a-to-site-b
    ```

---

### **Phase 4: Verification and Routing**

Now, let's test if the tunnel is working and add the routes for the private networks.

1.  **Verify IPSec Status (on either server):**
    ```bash
    sudo ipsec status
    ```
    You should see output like this, showing the connection is `ESTABLISHED`:
    ```
    Security Associations (1 up, 0 connecting):
    site-a-to-site-b[1]: ESTABLISHED 17 seconds ago, 203.0.113.10[203.0.113.10]...198.51.100.20[198.51.100.20]
    ```

2.  **Verify GRE Tunnel (from Server A):**
    ```bash
    ping 10.0.0.2
    ```
    You should get a successful reply. This confirms the GRE tunnel is up and running. If this fails, the IPSec tunnel is likely the problem.

3.  **Add Routes for Private Networks:**
    The tunnel is up, but the servers don't know to send traffic for the remote LAN through it.

    **On Server A:**
    ```bash
    sudo ip route add 192.168.20.0/24 via 10.0.0.2 dev gre0
    ```

    **On Server B:**
    ```bash
    sudo ip route add 192.168.10.0/24 via 10.0.0.1 dev gre0
    ```

4.  **Final Test:**
    From a machine on the `192.168.10.0/24` network, try to ping a machine on the `192.168.20.0/24` network. Or, from Server A, ping the private IP of Server B's LAN interface (if it has one).
    ```bash
    # From Server A
    ping 192.168.20.1
    ```

---

### **Phase 5: Making it Permanent (Netplan)**

The GRE tunnel and routes will disappear on reboot. We'll use `netplan` to make them permanent.

1.  **Find your netplan config file:**
    ```bash
    ls /etc/netplan/
    ```
    It will be named something like `01-netcfg.yaml` or `50-cloud-init.yaml`.

2.  **Edit the file on Server A:**
    ```bash
    sudo nano /etc/netplan/01-netcfg.yaml
    ```
    Add a `tunnels` section and a `routes` section. **Be careful with YAML indentation!**

    ```yaml
    network:
      version: 2
      # ... your existing eth0 or enp0s3 config ...
      tunnels:
        gre0:
          mode: gre
          local: 203.0.113.10
          remote: 198.51.100.20
          addresses: [10.0.0.1/30]
          ttl: 255
      routes:
        - to: 192.168.20.0/24
          via: 10.0.0.2
    ```

3.  **Edit the file on Server B:**
    ```bash
    sudo nano /etc/netplan/01-netcfg.yaml
    ```
    Add the corresponding configuration:

    ```yaml
    network:
      version: 2
      # ... your existing eth0 or enp0s3 config ...
      tunnels:
        gre0:
          mode: gre
          local: 198.51.100.20
          remote: 203.0.113.10
          addresses: [10.0.0.2/30]
          ttl: 255
      routes:
        - to: 192.168.10.0/24
          via: 10.0.0.1
    ```

4.  **Apply the Netplan Configuration on Both Servers:**
    ```bash
    sudo netplan apply
    ```

Your GRE over IPSec tunnel is now fully configured and will automatically come up on boot.



---

# freeIPA 

Below is a **complete, copy-paste-ready run-book** that takes you from **zero servers** to the **same enterprise-grade DevOps workflow** we described:  
centralized auth, sudo, 2FA, cert life-cycle, AD trust, and **Ansible-driven** host onboarding.

Time needed: **≈ 1 day** (most of it unattended install).

---

### 1. Design & prerequisites

| Item | Value |
|------|-------|
| **Linux dist** | RHEL 9 / Alma 9 / Oracle 9 (works on Ubuntu 22, commands differ slightly) |
| **IPA realm** | `EXAMPLE.CORP` |
| **IPA domain** | `ipa.example.corp` |
| **Servers** | 3 × VM (2 vCPU, 4 GB) for IPA replicas; N × client VMs |
| **DNS** | IPA runs **integrated BIND** (easiest); or delegate zone from corporate DNS |
| **Firewall** | open **80, 443, 88, 464, 389, 636, 123 udp, 53 tcp/udp** |
| **AD trust** (optional) | Windows 2016+ functional level; network reachability to AD DCs |

---

### 2. Build the first IPA server (30 min interactive)

```bash
# 1.1  OS prep
sudo dnf update -y
sudo hostnamectl set-hostname ipa1.example.corp
echo "192.168.10.11 ipa1.example.corp ipa1" | sudo tee -a /etc/hosts

# 1.2  install + firewalld
sudo dnf install -y @idm:DL1 ipa-server ipa-server-dns
sudo systemctl enable --now firewalld
sudo firewall-cmd --permanent --add-service=freeipa-4
sudo firewall-cmd --reload

# 1.3  run the installer
sudo ipa-server-install --setup-dns --auto-forwarders \
  --realm EXAMPLE.CORP --domain ipa.example.corp \
  --auto-reverse --mkhomedir --no-ntp
# supply passwords for Directory Manager and admin
# answer YES at the end
```

**Result**:  
Kerberos KDC, LDAP, Dogtag CA, BIND, OCSP, CRL, NTP all running on one box.

---

### 3. Add two replicas (high-availability)

```bash
# on replica candidate (ipa2.example.corp)
sudo dnf install -y @idm:DL1 ipa-server
sudo ipa-replica-install --setup-dns --auto-forwarders \
  --auto-reverse --mkhomedir --principal admin@EXAMPLE.CORP
# enter admin password; installer copies data from ipa1
```

Repeat for **ipa3**.  
You now have **3-way multi-master** for LDAP, Kerberos, CA.

---

### 4. Create the DevOps **identity scaffold** (5 min)

```bash
kinit admin   # obtain admin ticket (24 h)

# 4.1  POSIX groups
ipa group-add --desc='All developers' dev
ipa group-add --desc='App1 product' app1
ipa group-add --desc='App2 product' app2
ipa group-add --desc='Can sudo everything' ops

# 4.2  HBAC rule: only dev may SSH to staging
ipa hbacrule-add --desc='Dev staging access' dev-staging
ipa hbacrule-add-user --users=dev dev-staging
ipa hbacrule-add-host --hostgroups=staging dev-staging
ipa hbacrule-add-service --hbacsvcs=sshd dev-staging

# 4.3  sudo rules (stored in LDAP, pushed by SSSD)
ipa sudorule-add --desc='App1 can restart service' app1-restart
ipa sudorule-add-user --groups=app1 app1-restart
ipa sudorule-add-host --hostgroups=app1-servers app1-restart
ipa sudorule-add-allow-command --sudocmds='/usr/bin/systemctl restart *' app1-restart

# 4.4  enable 2FA (OTP) for prod
ipa config-mod --user-auth-type=otp --user-auth-type=password
ipa hbacrule-add prod-otp-only
ipa hbacrule-add-user --groups=ops prod-otp-only
ipa hbacrule-add-host --hostgroups=prod prod-otp-only
ipa hbacrule-add-service --hbacsvcs=sshd prod-otp-only
```

---

### 5. Add users & tokens (GUI or CLI)

```bash
ipa user-add alice --first=Alice --last=Lee --email=alice@corp.com --password
ipa group-add-member dev --users=alice
ipa otptoken-add alice   # prints QR code for Google-Auth
```

Alice now runs:
```bash
kinit alice   # prompted for password + OTP
ssh app1-staging01   # ticket forwarded, no password again
```

---

### 6. Optional – **one-way trust** to corporate AD

```bash
# on IPA master (network reachability to AD DCs)
ipa trust-add --type=ad CORP.EXAMPLE.COM \
  --admin adadmin --password
# enter AD domain-admin password
# IPA automatically creates cross-realm Kerberos trust
```

Windows users can now:
```
ssh -K alice@corp.example.com@ipa-host01
```
using **same password**, **no password sync**.

---

### 7. Client enrolment – **Ansible-as-code**

Create role `ipa-client` (excerpt):

```yaml
---
- name: install IPA client packages
  yum: name=ipa-client state=present

- name: enrol host to IPA
  command: |
    ipa-client-install --unattended --mkhomedir \
      --server ipa1.example.corp --domain ipa.example.corp \
      --realm EXAMPLE.CORP --principal enrollment@EXAMPLE.CORP \
      --password {{ vault_enrollment_otp }} --force-join
  register: out
  changed_when: "'Client configuration complete' in out.stdout"

- name: allow sssd to create home dirs
  lineinfile: path=/etc/pam.d/system-auth
              regexp='^session.*optional.*pam_mkhomedir'
              line='session optional pam_mkhomedir.so umask=0077'
```

Playbook used by **GitLab-CI**:

```yaml
# .gitlab-ci.yml snippet
provision-vm:
  script:
    - ansible-playbook -i aws_ec2.yml site.yml --tags ipa-client
```

**Result**: VM boots → pipeline enrolls it → HBAC & sudo rules **immediately active**.

---

### 8. Certificates for micro-services – **auto-renew**

```bash
# on host
ipa service-add HTTP/web-shop@EXAMPLE.CORP
# certmonger tracks the cert
ipa-getcert request -r -f /etc/pki/tls/certs/web-shop.crt \
                    -k /etc/pki/tls/private/web-shop.key \
                    -K HTTP/web-shop@EXAMPLE.CORP
```

Dogtag CA issues **90-day cert**; **certmonger renews at 60 days**, **reloads nginx** via hook.

---

### 9. Day-2 ops cheat-sheet

| Task | Command |
|------|---------|
| **List active tickets** | `klist` |
| **Disable user instantly** | `ipa user-disable bob` |
| **Extend prod access** | `ipa hbacrule-add-user prod-otp-only --users=alice && at now + 4 hours` |
| **Audit log** | `journalctl -u sssd --since "1 hour ago"` or `/var/log/krb5kdc.log` |
| **Promote new replica** | `ipa-replica-install on new VM` |
| **Backup IPA** | `ipa-backup --data --logs` (online, no downtime) |

---

### 10. Security hardening quick wins
- `ipa config-mod --user-auth-type=otp --user-auth-type=password`  
- `ipa pwpolicy-mod --minlength=14 --maxfail=5 --lockouttime=300`  
- Disable anonymous LDAP: `ldapmodify … nsslapd-allow-anonymous-access=never`  
- Enable **TLS 1.3 only** in `/etc/crypto-policies/config` → `update-crypto-policies --set FUTURE`  

---

### 🎉 Outcome
Follow the 10 steps above and you have:

✅ **Single identity** across 3 000+ Linux boxes  
✅ **SSH + sudo** managed from Git, not `/etc/passwd`  
✅ **2FA/OTP** on production, **Kerberos SSO** everywhere else  
✅ **AD trust** – Windows users keep one password  
✅ **Zero-touch** VM enrolment via Ansible  
✅ **Short-lived certs** auto-issued & renewed  

Copy, paste, adjust names, and you’ll reproduce the **enterprise DevOps workflow** in your own lab or data-centre.

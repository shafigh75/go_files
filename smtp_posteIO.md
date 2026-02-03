# setting up SMTP server

---

#### Question:
i have a server with a valid public IP lets say 91.12.26.18 and it is ubuntu server
i also have a valid domain which is on hostinger and its records are there and i have access to it lets say 
myapp.net 

now i want to setup a valid SMTP server to send emails from my domain and it must be valid and no spam and all so it must be a production and valid setup which easily can send emails to our users. 

give me the easiest and most straight forward step by step guide to do so? we can use any tool but it must be reliable and the easiest to setup with no extra hassle 

my target is to send emails from an account like lets say info@myapp.net 

---

This is a complex task because sending emails that land in the Inbox (and not Spam) relies heavily on **Reputation** and strict **DNS configuration**, not just installing software.

For the **easiest, most reliable, and straight-forward** production setup on Ubuntu, I recommend using **Poste.io**.

**Why Poste.io?**
Instead of manually installing 10 different packages (Postfix, Dovecot, SpamAssassin, ClamAV, Amavis, Nginx, etc.) and configuring text files for days, Poste.io is a complete mail server solution running inside a single **Docker** container. It includes a beautiful admin panel, auto-HTTPS (Let's Encrypt), and built-in spam filtering.

Here is your step-by-step guide.

---

### Phase 1: Prerequisites & Server Prep

**Crucial Warning:** Before proceeding, ensure your ISP or VPS provider allows traffic on **Port 25**. Many providers (AWS, Google Cloud, Azure, DigitalOcean) block Port 25 by default to prevent spam. You may need to request them to unblock it.

#### 1. Set the Hostname
We will use `mail.myapp.net` as the server's hostname. Login to your Ubuntu server via SSH:

```bash
sudo hostnamectl set-hostname mail.myapp.net
```

Open your hosts file:
```bash
sudo nano /etc/hosts
```
Add this line right below localhost:
```text
91.12.26.18 mail.myapp.net mail
```

#### 2. Configure Reverse DNS (rDNS/PTR) - **CRITICAL**
This is the #1 reason emails go to spam. You cannot do this inside Ubuntu; you must do this in your **Server Provider's Control Panel**.
*   Find the settings for your IP (`91.12.26.18`).
*   Look for "Reverse DNS", "PTR Record", or "rDNS".
*   Set the value to: `mail.myapp.net`
*   *If your rDNS does not match your hostname, Gmail/Yahoo will reject your emails.*

#### 3. Update System & Install Docker
We need Docker to run the mail server cleanly.

```bash
# Update repositories
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Start Docker
sudo systemctl enable docker
sudo systemctl start docker
```

---

### Phase 2: DNS Configuration (Hostinger)

Go to your **Hostinger Domain Management** console for `myapp.net` -> **DNS / Zone Editor**.
Add/Edit the following records. (Replace `91.12.26.18` with your actual IP).

#### 1. The A Record (Points the name to the IP)
*   **Type:** `A`
*   **Name:** `mail`
*   **Content:** `91.12.26.18`
*   **TTL:** 14400

#### 2. The MX Record (Tells the world who handles mail)
*   **Type:** `MX`
*   **Name:** `@`
*   **Mail Server:** `mail.myapp.net`
*   **Priority:** `10`
*   **TTL:** 14400

#### 3. The SPF Record (Authorizes your IP to send email)
*   **Type:** `TXT`
*   **Name:** `@`
*   **Content:** `v=spf1 mx ip4:91.12.26.18 -all`
*   *Note: If you already have an SPF record, edit it to include `ip4:91.12.26.18`.*

#### 4. The DMARC Record (Tells Gmail what to do if validation fails)
*   **Type:** `TXT`
*   **Name:** `_dmarc`
*   **Content:** `v=DMARC1; p=none; rua=mailto:admin@myapp.net`
*   *Note: We start with `p=none` (monitor mode) to ensure delivery while testing. Later change to `p=quarantine`.*

---

### Phase 3: Install and Run the Mail Server

Now, we launch the server. This command will download the mail server image, set up the data folders, and start it.

```bash
# Create a folder for mail data
sudo mkdir -p /opt/poste/data

# Run the server (Make sure ports 25, 80, 443, 110, 143, 465, 587, 993, 995 are free)
sudo docker run -d \
    --net=host \
    -e TZ=UTC \
    -v /opt/poste/data:/data \
    --name "mailserver" \
    -h "mail.myapp.net" \
    -t analogic/poste.io
```

*   `--net=host`: Uses the server's network stack directly (easiest for managing external IPs).
*   `/opt/poste/data`: This is where your emails and logs will be stored.
*   It handles automatic **SSL (Let's Encrypt)** setup for `mail.myapp.net` automatically upon startup.

---

### Phase 4: Final Configuration & DKIM

#### 1. Access the Admin Panel
Wait about 2-3 minutes for the container to start.
Open your web browser and go to:
**https://mail.myapp.net**

*   You might see a browser warning initially if the SSL certificate is still generating. Give it a moment or proceed.
*   You will see the **Poste.io Installer**.
*   **Server name:** `mail.myapp.net`
*   **Administrator email:** `info@myapp.net` (This creates your email account).
*   **Password:** Set a strong password.

#### 2. Configure DKIM (The final piece of the puzzle)
Once logged in to the dashboard:
1.  Go to **Virtual Domains** in the left menu.
2.  Click on `myapp.net`.
3.  Look for the **DKIM Key** section.
4.  Click "Generate new key".
5.  You will see a long text string (the public key).

**Go back to Hostinger DNS:**
*   **Type:** `TXT`
*   **Name:** `s20260203._domainkey` (The specific selector name Poste gives you, usually a date or 'default').
*   **Content:** `v=DKIM1; k=rsa; p=YOUR_LONG_KEY_HERE...`

*Tip: Poste.io usually tells you exactly what to put in the "Name" and "Content" fields.*

---

### Phase 5: Testing Your Production Setup

You now have a production-ready server. You must test the score to ensure you aren't marked as spam.

1.  **Send a Test Email:**
    *   Log in to the webmail at `https://mail.myapp.net/webmail` using `info@myapp.net`.
    *   Go to the website **[Mail-Tester.com](https://www.mail-tester.com/)**.
    *   It will give you a temporary email address (e.g., `test-abc@mail-tester.com`).
    *   Send an email from your new server to that address.
    *   **Subject:** "Hello from My App"
    *   **Body:** Write at least 2-3 sentences. Do not send "test test".

2.  **Check Score:**
    *   Click "Check your score" on Mail-Tester.
    *   **Goal:** You want a **10/10**.
    *   If it is lower, read the report. It will usually be missing DKIM or rDNS (Phase 1 and 4).

### Summary of Connection Details for your App
When configuring your application (Laravel, Django, Node.js, etc.) to send emails:

*   **Host:** `mail.myapp.net` (or `localhost` if the app is on the same server)
*   **Port:** `587` (TLS) or `465` (SSL)
*   **Encryption:** `TLS` or `SSL`
*   **Username:** `info@myapp.net`
*   **Password:** (The password you set in Phase 4)
*   **From Address:** `info@myapp.net`

### Maintenance
To upgrade the server in the future:
```bash
sudo docker stop mailserver
sudo docker rm mailserver
sudo docker pull analogic/poste.io
# Run the long docker run command from Phase 3 again
```

### Phase 6: Securing the Server (Firewall)

Since we used `--net=host` for ease of setup, your server is currently exposing all ports to the internet. We must use Ubuntu's firewall (UFW) to close everything except the specific mail ports.

**Warning:** Always allow SSH (Port 22) first, or you will lock yourself out of your server.

Run these commands one by one:

```bash
# 1. Allow SSH (So you don't lose access)
sudo ufw allow 22/tcp

# 2. Allow Web traffic (For Admin panel, Webmail, and Let's Encrypt SSL)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp

# 3. Allow SMTP (Sending and Receiving mail)
# Port 25: Server-to-Server communication (Incoming mail)
# Port 587/465: Client-to-Server communication (Outlook/Phone sending mail)
sudo ufw allow 25/tcp
sudo ufw allow 587/tcp
sudo ufw allow 465/tcp

# 4. Allow IMAP/POP3 (Reading mail on your phone/PC)
sudo ufw allow 143/tcp
sudo ufw allow 993/tcp
sudo ufw allow 110/tcp
sudo ufw allow 995/tcp

# 5. Enable the Firewall
sudo ufw enable
```
*Press `y` when it asks for confirmation.*

---

### Phase 7: Configure Email Clients (Outlook, iPhone, Thunderbird)

You don't always want to log in to the web browser to check email. You can add this account to your phone or desktop app.

Here are the generic settings you will need:

*   **Username:** `info@myapp.net`
*   **Password:** (The password you created in the Admin Panel)

**Incoming Server (IMAP):**
*   **Hostname:** `mail.myapp.net`
*   **Port:** `993`
*   **Security:** `SSL/TLS` (Normal password)

**Outgoing Server (SMTP):**
*   **Hostname:** `mail.myapp.net`
*   **Port:** `587` (Recommended) or `465`
*   **Security:** `STARTTLS` (for 587) or `SSL/TLS` (for 465)
*   **Authentication:** `Yes` (Use same settings as incoming)

---

### Phase 8: The "Warm-Up" Strategy (Avoid Spam Folder)

**This is the most important part for a new server.**

Even with 10/10 technical setup, Gmail and Outlook treat **new IPs** with suspicion. If you send 5,000 emails on Day 1, you will be blocked. You must "warm up" the IP to build reputation.

1.  **Week 1:** Send manually to people you know will reply.
    *   Send email to your personal Gmail, Yahoo, and Outlook accounts.
    *   **Reply** to those emails from your Gmail/Yahoo back to `info@myapp.net`.
    *   **Mark as "Not Spam"** if it lands in Junk.
    *   **Add to Contacts:** Add `info@myapp.net` to your personal Gmail contacts. This signals to Google that this is a legitimate sender.

2.  **Volume Schedule:**
    *   **Day 1-3:** Send < 50 emails/day.
    *   **Day 4-7:** Send < 100 emails/day.
    *   **Week 2:** Send < 500 emails/day.
    *   **Week 3:** Full production volume.

---

### Phase 9: Managing Users & Aliases

You likely need more than just `info`. Maybe you want `support` or `ceo`.

1.  Login to **https://mail.myapp.net/admin**
2.  Go to **Virtual Domains** -> click `myapp.net`.
3.  **To create a new mailbox:**
    *   Click "Create new account".
    *   Enter "support", give it a password, and set Quota.
4.  **To create an Alias (Forwarder):**
    *   *Example: You want `admin@myapp.net` to forward to `info@myapp.net` without a separate login.*
    *   Click "Create new alias".
    *   **Email:** `admin@myapp.net`
    *   **Goes to:** `info@myapp.net`

---

### Phase 10: Automatic Start & Backups

#### 1. Make the Server Auto-Start
If you reboot your Ubuntu server, Docker might not start the mail server automatically. Update the container configuration:

```bash
sudo docker update --restart=always mailserver
```

#### 2. Backup Strategy
All your emails, user accounts, and passwords are stored in `/opt/poste/data`. If you lose this folder, you lose everything.

To backup, you can simply stop the container and zip the folder:

```bash
# Stop server temporarily
sudo docker stop mailserver

# Create a zip backup with the date
sudo tar -czvf mail-backup-$(date +%F).tar.gz /opt/poste/data

# Start server again
sudo docker start mailserver
```
*Download that `.tar.gz` file to your local computer or upload it to S3/Google Drive for safekeeping.*

---

### Phase 11: Troubleshooting Common Issues

**Issue 1: I cannot send emails (Connection Time out)**
*   **Cause:** Your VPS provider (AWS, DigitalOcean, Vultr, Azure) is blocking Port 25.
*   **Test:** Run this command on your server: `telnet gmail-smtp-in.l.google.com 25`
*   **Result:** If it says "Trying..." and hangs, Port 25 is blocked.
*   **Fix:** Open a support ticket with your host: *"I have set up a mail server. Please unblock Port 25 for my IP 91.12.26.18 so I can send non-spam transactional emails."*

**Issue 2: My emails go to Gmail Spam folder**
*   **Cause:** Usually a missing PTR record or bad Content.
*   **Fix:**
    1. Check your IP at [MXToolbox Blacklist Check](https://mxtoolbox.com/blacklists.aspx) to see if the previous owner of your IP 91.12.26.18 was a spammer.
    2. Ensure your **PTR Record** (Reverse DNS) in your server control panel matches exactly `mail.myapp.net`.

**Issue 3: I can send, but I can't receive emails**
*   **Cause:** DNS Propagation or Firewall.
*   **Fix:**
    1. Check if you opened Port 25 in Phase 6.
    2. Check your MX Record in Hostinger. It must point to `mail.myapp.net`.
    3. Wait up to 24 hours for DNS changes to propagate globally.

---

#### FOLLOW UP QUESTION
ok one thing is that my server is on hetzner how can i configure the step you said: 
2. Configure Reverse DNS (rDNS/PTR) - CRITICAL

next thing is that on my server there is a webserver and some letsencrypt ssl already so my port 80 and 443 is bound which you said is critical to be open what can i do there? 

---

This changes the setup slightly, but it is actually a **better** architecture for a production server. We will put the Mail Server "behind" your existing Web Server (using a Reverse Proxy).

Here is the specific solution for **Hetzner** and your **Port Conflict**.

---

### Part 1: Configuring Reverse DNS on Hetzner

Hetzner makes this very easy, but the location depends on which Hetzner product you are using (Cloud vs. Dedicated).

**If you are using Hetzner Cloud (VPS):**
1.  Log in to the **Hetzner Cloud Console** (console.hetzner.cloud).
2.  Select your Project and click on your **Server**.
3.  Click on the **Networking** tab (top menu).
4.  Scroll down to the **Reverse DNS** section.
5.  You will see your IPv4 address (`91.12.26.18`).
6.  Click inside the "Reverse DNS" text box (it usually says generic text like `static.91.12...`).
7.  Change it to: `mail.myapp.net`
8.  Click **Save** (disk icon) or press Enter.

**If you are using Hetzner Robot (Dedicated Server):**
1.  Log in to **Hetzner Robot**.
2.  Go to **Server** -> Click on your server IP.
3.  Click on the **IPs** tab.
4.  Click on the text box next to your IP address under "Reverse DNS".
5.  Enter `mail.myapp.net` and click the Update button.

*Note: It may take 10-60 minutes for this change to propagate globally.*

---

### Part 2: Solving the Port 80/443 Conflict

Since your existing web server (Nginx or Apache) is using ports 80 and 443, we cannot give those ports to the Mail Server.

**The Strategy:**
1.  We will run the Mail Server's admin panel on a hidden internal port (e.g., `8080`).
2.  We will keep your specific Mail Ports (`25`, `587`, etc.) open to the public.
3.  We will tell your existing Web Server to handle the SSL and forward traffic for `mail.myapp.net` to the hidden port.

#### Step A: Run the Docker Container (Modified Command)

**Do NOT** run the command from the previous message. Use this modified command instead.
We are replacing `--net=host` with specific port mappings (`-p`) to avoid crashing your website.

```bash
# Stop any previous attempts
sudo docker stop mailserver
sudo docker rm mailserver

# Run the modified version
sudo docker run -d \
    -p 25:25 \
    -p 110:110 \
    -p 143:143 \
    -p 587:587 \
    -p 993:993 \
    -p 995:995 \
    -p 465:465 \
    -p 127.0.0.1:8080:80 \
    -e TZ=UTC \
    -e HTTPS=OFF \
    -v /opt/poste/data:/data \
    --name "mailserver" \
    -h "mail.myapp.net" \
    -t analogic/poste.io
```

**What changed?**
*   `-p 127.0.0.1:8080:80`: We mapped the mail server's web interface to port `8080` ONLY on localhost. The outside world cannot see it yet.
*   `-e HTTPS=OFF`: We tell Poste.io "Don't try to generate SSL certs," because your main server will handle that.

#### Step B: Generate SSL for the Mail Subdomain

Since Poste.io isn't doing it, you need to generate a Let's Encrypt certificate for `mail.myapp.net` on your host machine using Certbot.

```bash
# Assuming you use Nginx (Run this to get the cert)
sudo certbot certonly --nginx -d mail.myapp.net
```
*(If you use Apache, use `--apache` instead of `--nginx`)*.

#### Step C: Configure Nginx Reverse Proxy

Now create a configuration file to make `mail.myapp.net` accessible.

1. Create a new Nginx config file:
```bash
sudo nano /etc/nginx/sites-available/mail.myapp.net
```

2. Paste this configuration (Ensure you update the SSL paths if they differ):

```nginx
server {
    listen 80;
    server_name mail.myapp.net;
    return 301 https://$host$request_uri; # Force HTTPS
}

server {
    listen 443 ssl http2;
    server_name mail.myapp.net;

    # SSL Certificates (Check these paths match what Certbot created)
    ssl_certificate /etc/letsencrypt/live/mail.myapp.net/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mail.myapp.net/privkey.pem;

    # Security headers
    ssl_protocols TLSv1.2 TLSv1.3;

    location / {
        # Forward traffic to the Docker container on port 8080
        proxy_pass http://127.0.0.1:8080;
        
        # Required headers for Poste.io to know it's behind a proxy
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Websocket support (optional but good for webmail)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
    
    # Increase upload size for large email attachments
    client_max_body_size 50M;
}
```

3. Enable the site and restart Nginx:
```bash
sudo ln -s /etc/nginx/sites-available/mail.myapp.net /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

### Summary of Result
1.  **Hetzner:** Your IP `91.12.26.18` now officially identifies as `mail.myapp.net` via rDNS.
2.  **Web Traffic:**
    *   `myapp.net` -> Hits your existing website (unchanged).
    *   `mail.myapp.net` -> Hits Nginx -> Nginx forwards internally to Docker Port 8080 -> You see the Mail Admin Panel.
3.  **Email Traffic:**
    *   Incoming mail (Port 25) goes straight to the Docker container (mapped directly).
    *   Outlook/iPhone connections (Port 587/993) go straight to the Docker container.

### Important Note for Hetzner Users
**Hetzner blocks Port 25 (Outgoing) by default for new accounts.**
Even if you set everything up perfectly, your emails might not leave the server.

1.  Log in to the **Hetzner Cloud Console**.
2.  Go to your account settings or support section.
3.  Check if you have a limit on Port 25.
4.  If blocked, open a **Support Ticket**.
    *   **Subject:** Unblock Port 25
    *   **Message:** "Hello, I have setup a mail server on my VPS (IP: 91.12.26.18) for my business domain myapp.net. I have configured Reverse DNS and SPF/DKIM correctly. Please unblock Port 25 so I can send transactional emails."
    *   *They usually unblock it within 2 hours if your account is more than a few weeks old.*


---

#### FOLLOW UP QUESTION
ok another problem is that i already setup my mail for domain in the hostinger. does it not create a conflict? 


Yes, this creates a **major conflict**.

In the world of email, a domain (like `myapp.net`) can only have **one** primary "Post Office" that receives mail.

*   **MX Records** tell the world: "When you send email to `myapp.net`, deliver it here."
*   Currently, your MX records point to Hostinger.
*   In my guide, I asked you to point MX records to your VPS (`91.12.26.18`).

You cannot have both active for receiving mail at the same time. You have to make a choice between **Option A** (Switch fully to your VPS) or **Option B** (Hybrid/Advanced).

Since you asked for a "valid SMTP server" and a "production setup" to send emails, **Option A is highly recommended**. It gives you full control and saves you money (you stop paying Hostinger for email).

---

### Option A: The Full Switch (Recommended)
**You want your VPS to handle everything: Sending AND Receiving.**
This is the cleanest setup. Your VPS becomes the master.

**Action Required:**
1.  **Backup Old Emails:** If you have important emails currently in your Hostinger Webmail, back them up or forward them elsewhere. Once we change the DNS, you might lose access to that inbox depending on your hosting plan.
2.  **Clean Hostinger DNS:**
    *   Go to Hostinger DNS Zone Editor.
    *   **DELETE** any existing `MX` records that point to `mx1.hostinger.com` or similar.
    *   **DELETE** any existing `TXT` / SPF records that look like `v=spf1 include:_spf.mail.hostinger.com...`.
3.  **Add Your New Records:**
    *   Add the `MX` record pointing to `mail.myapp.net` (as described in Phase 2).
    *   Add the `SPF` record for your IP `91.12.26.18`.

**Result:**
*   You send email: It comes from your VPS (Valid).
*   Someone replies: It arrives at your VPS (Poste.io Inbox).
*   Hostinger is no longer involved in your email.

---

### Option B: The "Sender Only" Setup (Hybrid)
**You want to KEEP reading emails on Hostinger, but use the VPS to SEND automated emails (like password resets, newsletters).**

This is possible but tricky. If you configure `myapp.net` inside Poste.io, the server thinks "I own this domain."
*   *The Trap:* If your App (on the VPS) sends an email to `info@myapp.net`, the VPS will look inside itself, see the user, and drop the email in the local Docker container. **It will never send it to Hostinger.** You will check your Hostinger inbox and wonder why the email never arrived.

**To do this safely, you must update your DNS carefully:**

1.  **Keep Hostinger MX:** Do **NOT** change the MX records. Leave them pointing to Hostinger.
2.  **Update SPF (Merge):** You must authorize **BOTH** Hostinger and your VPS to send mail.
    *   Edit your `TXT` record (@).
    *   Change it to: `v=spf1 include:_spf.mail.hostinger.com ip4:91.12.26.18 -all`
    *   *(This says: "Hostinger is allowed to send, AND my VPS IP is allowed to send.")*
3.  **Use a different Subdomain (Smart Move):**
    *   Instead of sending from `info@myapp.net`, configure your VPS to send from `no-reply@mail.myapp.net` or `alerts@myapp.net`.
    *   In Poste.io, make sure you set the routing correctly so it knows to deliver non-local emails to the internet, not internally.

---

### My Recommendation
Since you are setting up **Poste.io** (a full-featured mail suite), **Option A (Full Switch)** is the correct path. Running a heavy mail server just to act as a "sender" while hosting mail elsewhere is overkill and confusing to manage.

**To proceed with the Full Switch:**
1.  Go to Hostinger DNS.
2.  **Delete** the old Hostinger MX and SPF records.
3.  **Add** the new records I gave you in Phase 2.
4.  Proceed with the Hetzner/Docker setup I gave you in the previous message.


---

#### FOLLOW UP QUESTION
ok in option B, you said we can do this with a subdomain instead of the main domain? i have full control over hostinger dns records so i can add any subdomain. does it fix the problem totally? how? give me step by step guide 

Yes, using a **dedicated subdomain** (like `mail.myapp.net` or `alerts.myapp.net`) is an excellent strategy.

**Does it fix the conflict totally?**
**Yes.** This creates a complete separation.
*   **Incoming Personal Mail:** Goes to Hostinger (`... @myapp.net`).
*   **Outgoing Transactional Mail:** Comes from your VPS (`... @mail.myapp.net`).

Because they use different names, the MX records, SPF records, and Reputation are treated separately by the internet. Your VPS will never accidentally "eat" an email meant for your Hostinger inbox.

Here is the **Step-by-Step Guide for the Subdomain Strategy**.

---

### Phase 1: Decide on the Subdomain
We need a name for this new "sending" system.
*   **Hostname:** `mail.myapp.net` (This is the name of the server).
*   **Sending Domain:** `mail.myapp.net` (Emails will come from `info@mail.myapp.net`).

*Note: If you prefer emails to come from `info@alerts.myapp.net`, just replace "mail" with "alerts" in the DNS steps below.*

---

### Phase 2: Hostinger DNS Configuration
Go to the **Hostinger DNS Zone Editor** for `myapp.net`. You need to add records specifically for the *subdomain*.

**1. Create the A Record (The Address)**
*   **Type:** `A`
*   **Name:** `mail`
*   **Value:** `91.12.26.18`
*   **TTL:** 14400

**2. Create the MX Record (Mail Routing for Subdomain)**
*   **Type:** `MX`
*   **Name:** `mail`
*   **Mail Server:** `mail.myapp.net`
*   **Priority:** `10`
*   *Note: This tells the internet "Emails sent to @mail.myapp.net belong to this server".*

**3. Create the SPF Record (Spam Protection for Subdomain)**
*   **Type:** `TXT`
*   **Name:** `mail`
*   **Value:** `v=spf1 mx ip4:91.12.26.18 -all`

**4. Create the DMARC Record (Reporting)**
*   **Type:** `TXT`
*   **Name:** `_dmarc.mail`
*   **Value:** `v=DMARC1; p=none; rua=mailto:admin@myapp.net`

---

### Phase 3: Server Preparation (Hetzner)
This ensures your server identifies itself correctly to avoid Spam folders.

**1. Set Hostname**
Run this on your VPS:
```bash
sudo hostnamectl set-hostname mail.myapp.net
```

**2. Set Reverse DNS (rDNS)**
*   Go to **Hetzner Console** -> Your Server -> **Networking** -> **Reverse DNS**.
*   Edit the entry for `91.12.26.18`.
*   Value: `mail.myapp.net` (Must match exactly).

---

### Phase 4: Install Mail Server (Poste.io) with Port Fix
Since you have a web server on Port 80/443, we use the "Hidden Port" method (Port 8080) + Nginx Proxy.

**1. Run the Docker Container**
```bash
# Stop any old containers
sudo docker stop mailserver 2>/dev/null
sudo docker rm mailserver 2>/dev/null

# Start the server
sudo docker run -d \
    -p 25:25 \
    -p 587:587 \
    -p 110:110 \
    -p 143:143 \
    -p 993:993 \
    -p 995:995 \
    -p 465:465 \
    -p 127.0.0.1:8080:80 \
    -e TZ=UTC \
    -e HTTPS=OFF \
    -v /opt/poste/data:/data \
    --name "mailserver" \
    -h "mail.myapp.net" \
    -t analogic/poste.io
```

**2. Setup Nginx Proxy & SSL**
(Only do this if you haven't done it in the previous step).

*   **Get SSL:** `sudo certbot certonly --nginx -d mail.myapp.net`
*   **Create Config:** `sudo nano /etc/nginx/sites-available/mail.myapp.net`
*   **Paste Config:**

```nginx
server {
    listen 80;
    server_name mail.myapp.net;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl;
    server_name mail.myapp.net;

    ssl_certificate /etc/letsencrypt/live/mail.myapp.net/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mail.myapp.net/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```
*   **Enable:** `sudo ln -s /etc/nginx/sites-available/mail.myapp.net /etc/nginx/sites-enabled/`
*   **Restart Nginx:** `sudo systemctl restart nginx`

---

### Phase 5: Final Configuration (Web Panel)

1.  Open your browser: `https://mail.myapp.net`
2.  **System Setup:**
    *   **Mailserver hostname:** `mail.myapp.net` (Do not put myapp.net)
    *   **Admin email:** `admin@mail.myapp.net` (or `info@mail.myapp.net`)
    *   **Password:** Create a strong one.
3.  Login.
4.  **Create DKIM Key (Crucial for Inbox delivery):**
    *   Go to **Virtual Domains** -> Click `mail.myapp.net`.
    *   Click **DKIM Key** -> Generate new key.
    *   It will give you a text string (e.g., `k=rsa; p=MIIBIjANBgkqh...`).
5.  **Add DKIM to Hostinger DNS:**
    *   Go back to Hostinger DNS.
    *   **Type:** `TXT`
    *   **Name:** `s20260203._domainkey.mail` (Replace `s20260203` with whatever "selector" Poste.io gave you. Note the `.mail` at the end!).
    *   **Value:** Paste the key from Poste.io.

---

### Phase 6: How to Use It

You now have a totally separate mail system.

**Your SMTP Settings for your App:**
*   **Host:** `mail.myapp.net`
*   **Port:** `587`
*   **Security:** `STARTTLS` (or TLS)
*   **Username:** `admin@mail.myapp.net`
*   **Password:** (Your password)
*   **From Address:** `admin@mail.myapp.net` (Must match the domain!)

**The Result:**
1.  Emails sent to `info@myapp.net` -> arrive at **Hostinger**.
2.  Emails sent from your App -> come from `admin@mail.myapp.net` via your **VPS**.
3.  **Zero Conflict.**

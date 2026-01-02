#!/bin/bash

# ==========================================
# Chimzy.com Production Server Setup Script
# ==========================================

# Configuration Variables
MAIN_DOMAIN="chimzy.com"
SUBDOMAINS=("backoffice" "n8n" "monitoring" "elastic" "minio" "shopify" "blog")
EMAIL="admin@chimzy.com" # CHANGE THIS to your real email for Let's Encrypt alerts
NGINX_CONFIG_PATH="/etc/nginx/sites-available/$MAIN_DOMAIN"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 1. Safety Checks
# ==========================================
if [ "$EUID" -ne 0 ]; then 
  echo -e "${RED}Please run as root (sudo ./setup_chimzy.sh)${NC}"
  exit 1
fi

echo -e "${YELLOW}!!! IMPORTANT !!!${NC}"
echo -e "Ensure that A-Records for ${GREEN}$MAIN_DOMAIN${NC} and all subdomains point to this server IP."
echo -e "If DNS is not propagated, Let's Encrypt will fail."
read -p "Press [Enter] to confirm DNS is set and continue..."

# 2. Update and Install Dependencies
# ==========================================
echo -e "${GREEN}Updating system and installing Nginx/Certbot...${NC}"
apt-get update -y
apt-get install -y nginx certbot python3-certbot-nginx

# 3. Create Directories and Default HTML
# ==========================================
echo -e "${GREEN}Setting up web directories and placeholder HTML...${NC}"

# Function to create site dir
setup_site_dir() {
    local domain=$1
    local dir="/var/www/$domain/html"
    
    mkdir -p "$dir"
    chown -R $USER:$USER "/var/www/$domain"
    chmod -R 755 "/var/www/$domain"

    # Create a simple index.html
    cat > "$dir/index.html" <<EOF
<!DOCTYPE html>
<html>
<head>
    <title>Welcome to $domain</title>
    <style>
        body { font-family: sans-serif; text-align: center; padding: 50px; background: #f4f4f4; }
        h1 { color: #333; }
        .container { background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); display: inline-block; }
    </style>
</head>
<body>
    <div class="container">
        <h1>$domain is live!</h1>
        <p>The backend services are currently being provisioned.</p>
    </div>
</body>
</html>
EOF
    echo "Created $dir"
}

# Setup Main Domain
setup_site_dir "$MAIN_DOMAIN"

# Setup Subdomains
for sub in "${SUBDOMAINS[@]}"; do
    setup_site_dir "$sub.$MAIN_DOMAIN"
done

# 4. Generate Nginx Configuration (Port 80 Only)
# ==========================================
echo -e "${GREEN}Generating Nginx configuration...${NC}"

# Start Config File
cat > "$NGINX_CONFIG_PATH" <<EOF
# Main Domain
server {
    listen 80;
    server_name $MAIN_DOMAIN;
    root /var/www/$MAIN_DOMAIN/html;
    index index.html;

    location / {
        try_files \$uri \$uri/ =404;
        
        # --- FUTURE BACKEND CONFIG (Uncomment when ready) ---
        # proxy_pass http://localhost:4040;
        # proxy_set_header Host \$host;
        # proxy_set_header X-Real-IP \$remote_addr;
        # proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        # proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

# Append Subdomain Configs
for sub in "${SUBDOMAINS[@]}"; do
    full_domain="$sub.$MAIN_DOMAIN"
    
    # Define specific backend ports or configs based on your previous dev setup
    # You can customize these defaults
    case $sub in
        "backoffice")
            PROXY_PORT="4050"
            EXTRA_CONFIG="
        # CORS Headers from Dev Setup
        # if (\$request_method = OPTIONS) {
        #    add_header 'Access-Control-Allow-Origin' 'https://$full_domain';
        #    add_header 'Access-Control-Allow-Methods' 'GET, POST, PUT, DELETE, OPTIONS';
        #    add_header 'Access-Control-Allow-Headers' 'Authorization,Content-Type';
        #    add_header 'Content-Length' 0;
        #    return 204;
        # }
        # add_header 'Access-Control-Allow-Origin' 'https://$full_domain' always;
            "
            ;;
        "n8n")
            PROXY_PORT="3344"
            EXTRA_CONFIG="" # Add specific n8n cors if needed
            ;;
        "shopify")
            PROXY_PORT="3000" # Assumptions based on dev
            EXTRA_CONFIG="" 
            ;;
        *)
            PROXY_PORT="8080" # Default placeholder port
            EXTRA_CONFIG=""
            ;;
    esac

    cat >> "$NGINX_CONFIG_PATH" <<EOF

# $sub
server {
    listen 80;
    server_name $full_domain;
    root /var/www/$full_domain/html;
    index index.html;

    location / {
        try_files \$uri \$uri/ =404;

        # --- FUTURE BACKEND CONFIG (Uncomment when ready) ---
        $EXTRA_CONFIG
        # proxy_pass http://localhost:$PROXY_PORT;
        # proxy_set_header Host \$host;
        # proxy_set_header X-Real-IP \$remote_addr;
        # proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        # proxy_set_header X-Forwarded-Proto \$scheme;
        # proxy_http_version 1.1;
        # proxy_set_header Upgrade \$http_upgrade;
        # proxy_set_header Connection "upgrade";
    }
}
EOF
done

# 5. Enable Site and Reload Nginx
# ==========================================
echo -e "${GREEN}Enabling site and reloading Nginx...${NC}"
ln -sf "$NGINX_CONFIG_PATH" /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default # Remove default nginx welcome page

nginx -t
if [ $? -eq 0 ]; then
    systemctl reload nginx
    echo -e "${GREEN}Nginx reloaded successfully.${NC}"
else
    echo -e "${RED}Nginx configuration failed. Please check errors.${NC}"
    exit 1
fi

# 6. Run Certbot (SSL)
# ==========================================
echo -e "${GREEN}Obtaining SSL Certificates...${NC}"

# Construct the domain list for certbot
DOMAIN_ARGS="-d $MAIN_DOMAIN"
for sub in "${SUBDOMAINS[@]}"; do
    DOMAIN_ARGS="$DOMAIN_ARGS -d $sub.$MAIN_DOMAIN"
done

# Run Certbot non-interactively
certbot --nginx $DOMAIN_ARGS --non-interactive --agree-tos --email "$EMAIL" --redirect

if [ $? -eq 0 ]; then
    echo -e "${GREEN}-------------------------------------------------------------${NC}"
    echo -e "${GREEN}SUCCESS! Your server is set up.${NC}"
    echo -e "Domains are live with HTTPS and pointing to static HTML pages."
    echo -e "To enable backends, edit: ${YELLOW}nano $NGINX_CONFIG_PATH${NC}"
    echo -e "Uncomment the proxy_pass sections when your apps are running."
    echo -e "${GREEN}-------------------------------------------------------------${NC}"
else
    echo -e "${RED}Certbot failed. Please check your DNS settings and try running certbot manually.${NC}"
fi

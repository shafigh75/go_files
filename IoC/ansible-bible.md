----------

# 📖 The Junior DevOps Ansible Bible

This handbook is designed as a production-ready, copy-paste-ready guide for Junior DevOps Engineers. It focuses on the most critical Ansible concepts and best practices, ensuring your automation is reliable, idempotent, and easy to maintain.

## I. 🌳 The Foundational Architecture

### 1.1. Core Concepts: The Agentless Advantage

Ansible works by connecting to managed nodes (servers) over standard SSH (Linux/Unix) or WinRM (Windows), executes its tasks, and then disconnects. **No agents are installed** on the managed nodes.

**Component**

**Role**

**Notes**

**Control Node**

Where Ansible is installed and playbooks are run.

Typically your workstation or a CI/CD server.

**Managed Nodes**

The target servers (host) to be configured.

Requires Python (for Linux) and SSH access.

**Inventory**

A list of managed nodes, organized into groups.

The **map** of your infrastructure.

**Playbook**

YAML files that define a set of automation steps (plays and tasks).

The core unit of execution.

**Module**

Small Python programs that perform a specific task (e.g., managing packages).

The **tools** Ansible uses to do work.

### 1.2. The Inventory File (`inventory.ini`)

The inventory is the most fundamental part of your Ansible setup.

#### Real-World Example: Static Inventory

Ini, TOML

```toml
# inventory.ini

# Group for web servers
[webservers]
web_node_01 ansible_host=192.168.1.10
web_node_02 ansible_host=192.168.1.11

# Group for database servers
[databases]
db_primary ansible_host=192.168.1.20
db_replica ansible_host=192.168.1.21

# A group of groups (all servers in the application)
[app_tier:children]
webservers
databases

# Variable for all hosts in the inventory
[all:vars]
ansible_user=devops_user
ansible_ssh_private_key_file=~/.ssh/devops_key

```

### 1.3. Ansible Configuration (`ansible.cfg`)

Always use a local `ansible.cfg` file in your project directory to define execution settings.

#### Production-Ready `ansible.cfg`

Ini, TOML

```toml
[defaults]
inventory = ./inventory.ini
remote_user = devops_user
# Only gather facts when explicitly requested (improves performance)
gathering = smart
# The roles directory relative to the config file
roles_path = ./roles
# Use cowsay, unless you are in a professional environment
nocows = True

[privilege_escalation]
# Standard settings for using 'sudo' on managed nodes
become = True
become_method = sudo
become_user = root
become_ask_pass = False # Only set to True for testing

```

----------

## II. ✍️ Playbooks: The Automation Code

Playbooks are the heart of Ansible automation, written in YAML.

### 2.1. Basic Playbook Structure (`setup_web.yml`)

This playbook demonstrates an **idempotent** (re-runnable without unintended side effects) web server setup.

YAML

```yaml
---
# File: setup_web.yml

# --- Play 1: Target the webservers group ---
- name: Configure and deploy a basic NGINX web server
  hosts: webservers
  become: yes # Use 'sudo' (become root) for privileged operations

  # --- Tasks: The ordered list of actions to perform ---
  tasks:
    - name: Ensure NGINX is installed (idempotent)
      # 'package' module ensures the desired state (installed)
      ansible.builtin.package:
        name: nginx
        state: present
      # Notify the handler 'restart nginx' if the package was installed/updated
      notify: restart nginx

    - name: Copy the production NGINX configuration file
      # 'template' module allows for dynamic configuration via Jinja2
      ansible.builtin.template:
        src: files/nginx.conf.j2
        dest: /etc/nginx/nginx.conf
        owner: root
        group: root
        mode: '0644'
      # Notify a handler if the config file was changed
      notify: restart nginx

    - name: Ensure the NGINX service is running and enabled at boot
      # 'service' module manages the service state
      ansible.builtin.service:
        name: nginx
        state: started
        enabled: yes

  # --- Handlers: Actions that are only run when notified by a task ---
  handlers:
    - name: restart nginx
      # This task will only run once, even if multiple tasks notify it
      ansible.builtin.service:
        name: nginx
        state: restarted

```

### 2.2. Production Code Organization: Roles

Roles are the definitive way to organize automation content, making it reusable and easy to share.

**Directory**

**Purpose**

`tasks/`

Main set of execution tasks.

`handlers/`

Handlers executed via `notify`.

`defaults/`

Default variables (lowest precedence).

`vars/`

Role-specific variables (higher precedence).

`templates/`

Jinja2 template files (`.j2`).

`files/`

Static files (e.g., bash scripts, certs).

#### Running a Role

The `setup_web.yml` example above is typically replaced by calling a role:

YAML

```yaml
# File: site.yml (main playbook)

- name: Deploy the full application stack
  hosts: app_tier
  roles:
    - role: common # Role for base OS configuration
    - role: web_tier # Role for NGINX/Web configuration
    - role: db_tier # Role for Database setup

```

----------

## III. 🛠️ Custom Modules: Extending Ansible

When a built-in module doesn't exist for a specific, proprietary task (e.g., interacting with a custom internal API), you must create a custom module.

### 3.1. Development Environment Setup

1.  **Create a library directory:** Ansible will look for custom modules in the `library/` folder relative to your playbook, or in the configured `library` path.
    
2.  **Ensure Python is available:** Custom modules are usually written in Python.
    

Bash

```bash
# In your project root
mkdir library
# The module file will go here: library/custom_status_check.py

```

### 3.2. Custom Module Template (`library/custom_status_check.py`)

This is a minimal, production-ready Python template for a custom module that checks a service status.

Python

```python
#!/usr/bin/python
# -*- coding: utf-8 -*-

from ansible.module_utils.basic import AnsibleModule

# Required for modules that will run on older Python versions or systems without standard libraries
# try:
#     import requests
#     HAS_REQUESTS = True
# except ImportError:
#     HAS_REQUESTS = False


def run_module():
    """
    The main entry point for the module.
    """
    # Define the module arguments/parameters
    module_args = dict(
        service_name=dict(type='str', required=True),
        expected_status=dict(type='str', default='running', choices=['running', 'stopped']),
    )

    # Instantiate the AnsibleModule object
    module = AnsibleModule(
        argument_spec=module_args,
        supports_check_mode=True # Good practice: allow 'ansible-playbook --check'
    )

    # Access module parameters
    service_name = module.params['service_name']
    expected_status = module.params['expected_status']

    # --- Start Custom Logic ---
    
    # In a real module, you would call an API, check a file, or run a command.
    # For this example, we'll simulate a command check.
    
    # Check if the service is actually running (using the standard 'command' module logic)
    rc, out, err = module.run_command(f'/usr/bin/systemctl is-active {service_name}')
    
    current_status = out.strip()
    
    # Check for success/failure conditions
    if rc != 0:
        module.fail_json(msg=f"Failed to check service status for {service_name}: {err}")
    
    # The command returns "active" if running, "inactive" otherwise.
    status_is_running = (current_status == 'active')
    
    # Determine the result based on the expected state
    if expected_status == 'running' and status_is_running:
        # State is already as desired (idempotent success)
        result = dict(
            changed=False,
            msg=f"Service {service_name} is running as expected."
        )
    elif expected_status == 'stopped' and not status_is_running:
         result = dict(
            changed=False,
            msg=f"Service {service_name} is stopped as expected."
        )
    else:
        # The expected state does not match the current state.
        # Note: Custom modules typically don't change state, they report and fail/succeed.
        # We report failure if the expected status is not met.
        module.fail_json(msg=f"Service {service_name} status is '{current_status}', but expected '{expected_status}'.")

    # --- End Custom Logic ---

    # Exit the module successfully, returning the result dictionary
    module.exit_json(**result)


if __name__ == '__main__':
    run_module()

```

### 3.3. Using the Custom Module

You use your custom module exactly like a built-in module, using its file name without the `.py` extension.

YAML

```yaml
- name: Use the custom module to verify critical service status
  hosts: all
  tasks:
    - name: Check if 'monitoring_agent' is running
      # Calls the custom module: library/custom_status_check.py
      custom_status_check:
        service_name: monitoring_agent
        expected_status: running
      # The task will fail if the service is not running

```

----------

## IV. 🔒 Secrets Management: Ansible Vault

**Never store passwords, tokens, or private keys in plain text.** Ansible Vault is the standard, built-in solution for encrypting sensitive data.

### 4.1. Vault Commands

**Command**

**Purpose**

`ansible-vault create secret.yml`

Create a new encrypted file.

`ansible-vault encrypt my_file.txt`

Encrypt an existing file.

`ansible-vault edit group_vars/prod.yml`

Edit an encrypted file in your default text editor.

`ansible-vault view secret.yml`

View the decrypted contents.

`ansible-vault rekey secret.yml`

Change the vault password.

### 4.2. Recommended Usage: Encrypting `group_vars`

Encrypt your production variables file and include the sensitive data there.

Bash

```bash
# Encrypt the production variables file
ansible-vault encrypt group_vars/prod/vault.yml

```

#### Example Encrypted File (`group_vars/prod/vault.yml`)

YAML

```yaml
# Only visible after decryption
prod_db_password: "MySuperSecurePassword123"
aws_access_key: "AKIAIOSFODNN7EXAMPLE"

```

#### Example Playbook Usage

The variable is automatically decrypted at runtime:

YAML

```yaml
- name: Deploy application with secret environment variables
  hosts: webservers
  tasks:
    - name: Set environment variables using encrypted password
      ansible.builtin.template:
        src: files/app_config.j2
        dest: /etc/app/config.ini
      vars:
        # Variable 'prod_db_password' is automatically loaded and decrypted
        db_pass: "{{ prod_db_password }}" 

```

### 4.3. Automating Vault Decryption

In a production environment, you should use a **vault password file** instead of typing the password every time.

Bash

```bash
# Create a password file (ensure this file is very secure or outside source control!)
# Recommended: use a secure secrets manager (like HashiCorp Vault, AWS Secrets Manager)
# to fetch the password into this file at runtime in your CI/CD pipeline.
echo "MyVaultMasterPassword" > .vault_pass.txt 

# Run the playbook, referencing the password file
ansible-playbook site.yml --vault-password-file .vault_pass.txt

```

----------

## V. 🐛 Error Handling & Debugging

Robust playbooks are designed to handle failures gracefully.

### 5.1. Graceful Failure (`block`/`rescue`/`always`)

Use the `block` structure for transaction-like execution.

YAML

```yaml
- name: Attempt critical operation with guaranteed cleanup
  hosts: target_servers
  tasks:
    - name: Transaction block
      block:
        - name: 1. Deploy sensitive temporary file
          ansible.builtin.copy:
            content: "sensitive data"
            dest: /tmp/temp_data.txt
            
        - name: 2. CRITICAL TASK - Fails if external API is down
          ansible.builtin.uri:
            url: https://external.api/v1/deploy
            method: POST
            status_code: 200

      rescue:
        - name: FAILURE: Log the error to a central logging server
          ansible.builtin.debug:
            msg: "CRITICAL TASK FAILED! Cleanup must be performed."
        # Execution jumps here if any task in 'block' fails

      always:
        - name: ALWAYS: Ensure the temporary file is removed (cleanup)
          ansible.builtin.file:
            path: /tmp/temp_data.txt
            state: absent
          # This task runs regardless of success or failure in 'block'/'rescue'

```

### 5.2. Controlling Task Failure (`failed_when`)

You can override Ansible's default failure behavior (non-zero return code) for specific scenarios.

YAML

```yaml
- name: Run a cleanup script that sometimes fails but is safe to ignore
  ansible.builtin.command: /usr/local/bin/run_cleanup_script.sh
  # This task will ONLY fail if the exit code is > 2, otherwise it's considered successful
  failed_when: cleanup_script.rc > 2
  register: cleanup_script

```

### 5.3. Debugging Essentials

Always use the `debug` module to inspect variables and execution flow.

YAML

```yaml
- name: Register and inspect the result of a command
  ansible.builtin.command: cat /etc/os-release
  register: os_info

- name: DEBUG: Print the registered variable's contents
  ansible.builtin.debug:
    # Print the entire registered variable (rc, stdout, stderr, etc.)
    var: os_info 

- name: DEBUG: Print a single variable or fact
  ansible.builtin.debug:
    # Print a specific detail from Ansible Facts
    msg: "The target host's OS is: {{ ansible_distribution }} {{ ansible_distribution_version }}"

```

----------


## VI. 🎯 Advanced Control Flow and Jinja2 Templating

### 6.1. Conditionals: The `when` Statement

The `when` clause executes a task only if a given condition is met. This condition uses Jinja2 expressions.

#### Production-Ready Example: OS-Specific Configuration

YAML

```yaml
- name: Apply OS-specific package installation
  hosts: all
  become: yes

  tasks:
    - name: Install 'apache2' on Debian/Ubuntu systems
      ansible.builtin.package:
        name: apache2
        state: present
      # Condition: only run if the operating system is Debian or Ubuntu
      when: ansible_os_family == "Debian"

    - name: Install 'httpd' on RHEL/CentOS systems
      ansible.builtin.package:
        name: httpd
        state: present
      # Condition: only run if the operating system is RedHat
      when: ansible_os_family == "RedHat"
      
    - name: Handle multiple conditions (AND/OR)
      ansible.builtin.debug:
        msg: "This is a webserver running a modern kernel."
      when: 
        # Logical AND
        - "'webservers' in group_names" 
        - "ansible_kernel | version_compare('4.15', '>=')"

```

### 6.2. Loops: Running Tasks Iteratively

Loops are essential for repetitive tasks, such as creating multiple users or installing a list of packages.

#### Example: Looping Over a List (Installation)

YAML

```yaml
- name: Install a required list of packages
  hosts: all
  become: yes
  tasks:
    - name: Install common utilities
      ansible.builtin.package:
        name: "{{ item }}"
        state: present
      loop:
        - vim
        - git
        - tmux
        - zip

```

#### Example: Looping Over a Dictionary (User Creation)

Using a dictionary allows you to define multiple properties per item in the loop.

YAML

```yaml
# In group_vars/all/users.yml
app_users:
  - name: application_admin
    uid: 1001
    groups: "sudo,www-data"
  - name: deployment_user
    uid: 1002
    groups: "deployment"

# In the playbook tasks
- name: Create users with specific UIDs and groups
  hosts: all
  become: yes
  tasks:
    - name: Create user {{ item.name }}
      ansible.builtin.user:
        name: "{{ item.name }}"
        uid: "{{ item.uid }}"
        groups: "{{ item.groups }}"
        state: present
      # Use `loop` to iterate over the dictionary variable
      loop: "{{ app_users }}"

```

### 6.3. Jinja2 Templating Deep Dive

Jinja2 is the templating language used to dynamically generate configuration files (`.j2` extension).

#### Example Template: `files/app_config.ini.j2`

Code snippet

```
# Application Configuration File - Generated by Ansible
[General]
environment = {{ env_name | default('staging') }}
log_level = {{ log_level_var }}

[Database]
# Filter Example: convert to lowercase and replace spaces
db_host = {{ ansible_hostname }}.db.internal
db_port = 5432
# Vault Example: use a filter to base64 encode the secret
db_password = {{ vault_db_password | b64encode }}

```

#### Example Playbook Task: Deploying the Template

YAML

```yaml
- name: Deploy the configuration file using the template
  hosts: webservers
  vars:
    # Example playbook/task-level variable definition
    log_level_var: INFO 
  tasks:
    - name: Template and deploy application config
      ansible.builtin.template:
        src: files/app_config.ini.j2
        dest: /etc/app/config.ini
        owner: appuser
        group: appuser
        mode: '0600'
      # If the file contents change after templating, notify the handler
      notify: reload application

```

----------

## VII. ⚙️ Essential Built-in Modules

Using the correct module is key to writing idempotent, readable playbooks. The `ansible.builtin` collection prefix is always recommended for core modules.

### 7.1. File Management (`copy`, `template`, `file`)

**Module**

**Purpose**

**Example (.yml)**

**`copy`**

Copies a static file from the control node to the managed node.

`yaml - name: Copy static logo file ansible.builtin.copy: src: files/logo.png dest: /var/www/html/logo.png owner: www-data group: www-data mode: '0644'`

**`template`**

Dynamically generates a file using Jinja2 and copies it.

(See section 6.3)

**`file`**

Manages file system objects (directories, files, links).

`yaml - name: Ensure logs directory exists ansible.builtin.file: path: /var/log/my_app state: directory owner: appuser group: appuser mode: '0755'`

### 7.2. Package and Service Management

**Module**

**Purpose**

**Example (.yml)**

**`package`**

Manages OS packages (abstracts `apt`, `yum`, `dnf`, etc.). **Always prefer this.**

`yaml - name: Ensure MariaDB is latest version ansible.builtin.package: name: mariadb-server state: latest`

**`service`**

Manages system services (abstracts `systemd`, `init.d`, etc.).

`yaml - name: Ensure firewall is running and enabled ansible.builtin.service: name: firewalld state: started enabled: yes`

### 7.3. Executing Commands (`shell` vs. `command`)

**Rule of Thumb: Always use a dedicated Ansible module first.** Use `command` or `shell` only if a module does not exist, or if the logic is too complex for Jinja2.

-   **`command`**: Executes a basic command _without_ involving a shell (no pipes, redirection, or environment variables like `$HOME`). **More secure.**
    
-   **`shell`**: Executes a command through the shell (`/bin/sh`), allowing for shell features (pipes `|`, redirection `>`). **Necessary for complex commands, but less secure.**
    

#### Example: Using `command` (Preferred for simple execution)

YAML

```yaml
- name: Check server uptime (simple command)
  ansible.builtin.command: /usr/bin/uptime
  register: uptime_result

- name: Debug uptime
  ansible.builtin.debug:
    msg: "System uptime: {{ uptime_result.stdout }}"

```

#### Example: Using `shell` (Needed for complex/piped execution)

YAML

```yaml
- name: Compress web logs and delete old ones (requires shell piping and globbing)
  ansible.builtin.shell: |
    # Use | for readability (multi-line shell script)
    find /var/log/nginx -type f -name "*.log" -mtime +7 | xargs gzip;
    rm -f /var/log/nginx/*.log.old
  # Always register the result to inspect success/failure
  register: log_cleanup
  # Use 'changed_when' to explicitly control idempotency
  changed_when: log_cleanup.rc != 0 
  # Set a timeout for long-running scripts
  args:
    executable: /bin/bash # Ensure we use a predictable shell
    # Prevent this task from failing the play if the command returns a non-zero exit code
    warn: no 

```

----------

## VIII. 🏗️ The Production-Ready Project Structure

This structure leverages **Roles**, **Inventory Variables (`group_vars`)**, and **Ansible Vault** for a clean, secure, and scalable automation project.

### 8.1. Project Directory Structure

```bash
ansible-project/
├── ansible.cfg                          # 1. Project-specific Ansible settings
├── inventory.ini                        # 2. Static inventory file
├── site.yml                             # 3. Main entry point playbook (calls roles)
├── group_vars/                          # 4. Variables applied to groups
│   ├── all/
│   │   ├── common.yml                   # General non-sensitive variables for all hosts
│   │   └── vault.yml                    # Encrypted global variables (e.g., control node SSH key path)
│   └── prod/
│       └── app_settings.yml             # Non-sensitive variables only for the 'prod' group
├── roles/                               # 5. Reusable automation units
│   ├── base_os/
│   │   ├── tasks/
│   │   │   └── main.yml                 # Package installs, user setup
│   │   └── handlers/
│   │       └── main.yml                 # Restart services, etc.
│   └── web_server/
│       ├── tasks/
│       │   └── main.yml                 # NGINX install, config deployment
│       ├── templates/
│       │   └── nginx.conf.j2            # The dynamic configuration file
│       └── defaults/
│           └── main.yml                 # Role default variables (lowest precedence)
└── library/                             # 6. Custom Python modules (as shown in section III)
    └── custom_status_check.py

```

### 8.2. File Contents Showcase

#### 1. `ansible.cfg`

Ini, TOML

```toml
[defaults]
inventory = ./inventory.ini
roles_path = ./roles
# Use 'callback' to get nice output in the terminal
stdout_callback = yaml

```

#### 2. `inventory.ini`

Ini, TOML

```toml
[webservers]
web01
web02

[databases]
db01

[prod:children]
webservers
databases

```

#### 3. `site.yml` (The Orchestrator)

YAML

```yaml
---
# File: site.yml - The main playbook to deploy the entire application stack

- name: Apply base configuration to all servers
  hosts: all
  become: yes
  roles:
    - role: base_os

- name: Deploy the Web Tier (NGINX)
  hosts: webservers
  become: yes
  # Inject an environment-specific variable
  vars:
    app_env: Production
  roles:
    - role: web_server

```

#### 4. `roles/web_server/tasks/main.yml` (The Core Logic)

YAML

```yaml
---
# File: roles/web_server/tasks/main.yml

- name: Ensure NGINX is installed
  ansible.builtin.package:
    name: nginx
    state: present
  # The task is tagged for selective execution (e.g., ansible-playbook site.yml --tags install)
  tags: install

- name: Deploy NGINX configuration using template
  ansible.builtin.template:
    src: nginx.conf.j2
    dest: /etc/nginx/nginx.conf
    owner: root
    group: root
    mode: '0644'
  notify: restart nginx
  tags: config

- name: Ensure NGINX service is started and enabled
  ansible.builtin.service:
    name: nginx
    state: started
    enabled: yes
  tags: service

```

#### 5. `roles/web_server/templates/nginx.conf.j2` (Secure Templating)

Code snippet

```
# NGINX Configuration File (Templated)

# Variable passed from the site.yml vars block
user www-data;
worker_processes auto;

# Variable loaded from group_vars (e.g., prod/app_settings.yml)
pid /run/nginx-{{ app_port }}.pid;

events {
    worker_connections 768;
}

http {
    # Using the host's inventory name as the server name
    server {
        listen 80;
        server_name {{ inventory_hostname }}.local;
        root /var/www/{{ app_env | lower }};
        index index.html;
    }
}

```

This complete structure provides **modularity** (roles), **security** (vault), **flexibility** (Jinja2 templates), and a clear **entry point** (`site.yml`), making it a robust foundation for any DevOps automation task.

----------

This is the most crucial final step! Understanding how to write a custom module empowers you to automate anything, even proprietary internal tools.

I will now provide a comprehensive, real-world tutorial for creating a meaningful custom Python Ansible module, followed by an expansion of the most needed built-in modules.

----------

# 📖 The Junior DevOps Ansible Bible (Final Appendices)

## IX. 🐍 Real-World Custom Module Tutorial: Service Health Checker

We will create a custom module called `http_status_check` that queries an application's health endpoint (`/health`) and asserts that the HTTP response code is what we expect. This is a common requirement in CI/CD pipelines and deployment verification.

### Prerequisites

1.  **Python on Control Node:** Ansible is run in Python.
    
2.  **`requests` library:** We'll use the popular `requests` library for HTTP calls. While Ansible usually avoids external dependencies on managed nodes, for the control node's custom module logic, this is fine, provided the library is installed in your Ansible environment (`pip install requests`).
    

### 9.1. Module Setup and Boilerplate

Create the module file in your project's `library` directory:

Bash

```
# In your ansible-project/ directory
touch library/http_status_check.py

```

Paste the following basic structure into `library/http_status_check.py`:

Python

```python
#!/usr/bin/python
# -*- coding: utf-8 -*-

from __future__ import (absolute_import, division, print_function)
__metaclass__ = type

DOCUMENTATION = r'''
---
module: http_status_check
short_description: Check the HTTP status code of a given URL.
version_added: "1.0.0"
description:
    - This module attempts to connect to a URL and verifies the returned HTTP status code against an expected value.
options:
    url:
        description: The full URL (including http/https) of the endpoint to check.
        type: str
        required: true
    expected_status:
        description: The HTTP status code expected from the URL (e.g., 200, 301).
        type: int
        default: 200
author:
    - DevOps Team
'''

# Standard Ansible import for module development
from ansible.module_utils.basic import AnsibleModule

# Import required external libraries
try:
    import requests
    HAS_REQUESTS = True
except ImportError:
    HAS_REQUESTS = False

def run_module():
    # Define the module arguments/parameters
    module_args = dict(
        url=dict(type='str', required=True),
        expected_status=dict(type='int', default=200),
    )

    # Instantiate the AnsibleModule object
    module = AnsibleModule(
        argument_spec=module_args,
        supports_check_mode=True,
    )

    # --- Start Custom Logic ---
    
    # Check for required dependencies
    if not HAS_REQUESTS:
        module.fail_json(msg="The 'requests' Python library is required for this module.")

    # Access module parameters
    url = module.params['url']
    expected_status = module.params['expected_status']
    
    # Initialize result dictionary
    result = dict(
        changed=False,
        original_message='',
        message='Success: Status code matched expected value.',
        url_checked=url,
        status_code=None
    )

    # Only run the actual check if not in check mode
    if not module.check_mode:
        try:
            # Perform the HTTP request
            response = requests.get(url, timeout=10)
            status_code = response.status_code
            result['status_code'] = status_code

            if status_code != expected_status:
                # Failure condition: Status code mismatch
                result['changed'] = True # Report a change (or a failure to meet state)
                module.fail_json(
                    msg=f"Expected status code {expected_status} but received {status_code}.",
                    **result
                )
            
        except requests.exceptions.RequestException as e:
            # Failure condition: Request failed (timeout, DNS error, connection refused)
            module.fail_json(
                msg=f"Request to {url} failed: {str(e)}",
                **result
            )

    # --- End Custom Logic ---

    # Exit the module successfully
    module.exit_json(**result)


if __name__ == '__main__':
    run_module()

```

### 9.2. Key Components Explained

1.  **`DOCUMENTATION`**: Crucial! This YAML block defines the module's behavior, parameters, and usage. Ansible uses this to generate the `ansible-doc` output.
    
2.  **Dependency Check (`HAS_REQUESTS`)**: The code explicitly checks for the `requests` library and immediately calls `module.fail_json()` if it's missing. This provides clear error feedback.
    
3.  **`module_args`**: Defines the module's contract (what inputs it takes). `required=True` ensures the user provides the `url`.
    
4.  **`module.check_mode`**: Ensures that the module doesn't perform destructive actions when running with `--check` or `-C`. Since our module is read-only, we only perform the HTTP request outside of check mode.
    
5.  **`module.fail_json(...)`**: The standard way to tell Ansible that the task failed. The play will stop (unless handled by `ignore_errors`).
    
6.  **`module.exit_json(...)`**: The standard way to tell Ansible the task succeeded. It accepts the final `result` dictionary.
    

### 9.3. Using the Custom Module in a Playbook

You can now use this module to verify deployments in a testing stage of your pipeline:

YAML

```yaml
---
# File: verify_deployment.yml

- name: Post-Deployment Verification Check
  hosts: verification_server
  gather_facts: no

  tasks:
    - name: Wait for the application to be fully up and return 200
      # Use the built-in 'wait_for_connection' first, then our custom module
      ansible.builtin.wait_for:
        port: 8080
        host: web01
        delay: 5
        timeout: 60
        state: started

    - name: Check application health endpoint for 200 OK status
      http_status_check: # Calls the custom module
        url: "http://web01:8080/health"
        expected_status: 200
      register: health_check_result

    - name: Debug the successful result from the custom module
      ansible.builtin.debug:
        msg: "Application is healthy! Received {{ health_check_result.status_code }} from {{ health_check_result.url_checked }}"

```

----------

## X. 🎯 Crucial Built-in Modules (Deep Dive)

While we covered the basics, these modules are the workhorses of complex DevOps playbooks and require more detailed examples.

### 10.1. `lineinfile` and `blockinfile`: Configuration Editing

These are essential for managing existing configuration files without overwriting the entire file (which `template` often does).

**Module**

**Purpose**

**Key Parameters**

**`lineinfile`**

Ensures a single line is present, absent, or replaced by matching a regex.

`path`, `state`, `line`, `regexp`

**`blockinfile`**

Inserts or replaces a multi-line block of text, marked by unique start/end markers.

`path`, `block`, `marker`

#### Example: Managing a Single Line (e.g., SSH config)

YAML

```yaml
- name: Ensure SSH PermitRootLogin is set to 'no'
  ansible.builtin.lineinfile:
    path: /etc/ssh/sshd_config
    # Regex to match: starts with 'PermitRootLogin' followed by any characters
    regexp: '^PermitRootLogin.*'
    # The replacement line
    line: 'PermitRootLogin no'
    # Creates the file if it doesn't exist (if create: yes)
    state: present
  notify: restart sshd

```

#### Example: Managing a Multi-line Block (e.g., Virtual Host)

YAML

```yaml
- name: Add a custom VirtualHost block to Apache config
  ansible.builtin.blockinfile:
    path: /etc/httpd/conf/httpd.conf
    # The multi-line block to insert
    block: |
      <VirtualHost *:80>
          ServerName app.{{ app_env }}.internal
          DocumentRoot /var/www/app_data
          ErrorLog /var/log/httpd/app_error.log
      </VirtualHost>
    # Unique marker to identify the block on subsequent runs
    marker: "# {mark} ANSIBLE MANAGED BLOCK FOR APP VHOST"
  notify: reload httpd

```

### 10.2. `uri`: Interacting with REST APIs

The `uri` module is used for making HTTP/S requests, crucial for interacting with cloud providers, secrets managers, or deployment APIs.

YAML

```yaml
- name: POST a message to a notification service upon successful deployment
  # This task relies on the successful execution of prior tasks
  ansible.builtin.uri:
    url: "https://api.internal.corp/notify"
    method: POST
    # Request body can be JSON, which is common for APIs
    body: 
      message: "Deployment of {{ app_name }} complete on {{ inventory_hostname }}"
      status: "success"
    body_format: json
    # Use Ansible Vault for the API key authentication
    headers:
      Authorization: "Bearer {{ notification_api_token }}"
    # Expect a standard 202 or 200 response
    status_code: [200, 202] 
  delegate_to: localhost # Run the API call from the control node
  run_once: true # Only run this task once, even if targeting multiple hosts

```

### 10.3. `git`: Managing Code Repositories

Essential for deploying applications and configuration from version control.

YAML

```yaml
- name: Deploy the application code from GitLab
  ansible.builtin.git:
    # URL (can contain vault variables for SSH keys or tokens)
    repo: '{{ git_repo_url }}'
    # Target directory on the managed node
    dest: /opt/application/{{ app_name }}
    # Ensure a specific tag/branch/commit is deployed (recommended for stability)
    version: '{{ app_version_tag | default("main") }}'
    # Only clone if the directory is missing (improves performance)
    clone: yes
    # If using SSH key authentication
    key_file: /home/appuser/.ssh/git_deploy_key
    # Ensures the remote branch/tag exists locally before updating
    update: yes 
  become_user: appuser # Run the git clone as the application user

```

### 10.4. `debug`: Advanced Variable Inspection

Beyond simple variable printing, `debug` can show complex object structures.

YAML

```yaml
- name: Show all variables available to the current host
  ansible.builtin.debug:
    # Use the 'vars' magic variable
    var: vars
    # Use the 'verbosity' option to only run this when '-vv' (very verbose) is passed
    verbosity: 2 

- name: Show the full content of the NGINX restart handler notification queue
  ansible.builtin.debug:
    # Use the 'ansible_run_handlers' magic variable
    var: ansible_run_handlers
    when: ansible_run_handlers is defined and ansible_run_handlers | length > 0

```

----------

# XI. 📦 More Production-Critical Modules

### 11.1. `archive` and `unarchive`: Compression and Decompression

These modules handle moving files between archives (like `.tar.gz`, `.zip`) and the filesystem. This is crucial for deploying large application bundles.

**Module**

**Purpose**

**Key Parameters**

**`archive`**

Creates a compressed archive from multiple files/directories on the **managed node**.

`path`, `dest`, `format`

**`unarchive`**

Extracts an archive file (`src`) to a target directory (`dest`) on the **managed node**.

`src`, `dest`, `remote_src`

#### Example: Deploying an Application Bundle

YAML

```
- name: Extract application package to deployment path
  hosts: webservers
  tasks:
    - name: Ensure the application package is extracted
      ansible.builtin.unarchive:
        # Source file path on the managed node
        src: /tmp/app_build_v1.2.tar.gz 
        # Destination directory for extraction
        dest: /opt/application/
        # Set to 'yes' if the archive is already on the managed node
        remote_src: yes 
        owner: appuser
        group: appuser

```

#### Example: Backing Up Log Files

YAML

```yaml
- name: Archive old logs before cleanup (on the managed node)
  ansible.builtin.archive:
    path: /var/log/app/*.log
    dest: /var/log/app_backup/{{ inventory_hostname }}-{{ ansible_date_time.iso8601_basic_short }}.tgz
    format: gz
    # Remove original files after successful archival
    remove: yes 

```

### 11.2. `yum_repository` / `apt_repository`: Managing Package Sources1

Before you can install packages (using the `package` module), you often need to ensure the correct software repository is configured (e.g., for custom or third-party packages).

**Module**

**Purpose**

**Key Parameters**

**`yum_repository`**

Manages RPM repositories on RHEL/CentOS systems.

`name`, `description`, `baseurl`, `gpgcheck`

**`apt_repository`**

Manages APT repositories on Debian/Ubuntu systems.

`repo`, `state`, `filename`

#### Example: Configuring an External APT Repository

YAML

```yaml
- name: Add NGINX stable repository key and source
  hosts: webservers
  when: ansible_os_family == "Debian"
  tasks:
    - name: Add NGINX official PGP key
      ansible.builtin.apt_key:
        url: https://nginx.org/keys/nginx_signing.key
        state: present

    - name: Add the NGINX stable repository source
      ansible.builtin.apt_repository:
        repo: 'deb http://nginx.org/packages/{{ ansible_distribution | lower }}/ {{ ansible_distribution_release }} nginx'
        state: present
        update_cache: yes # Run apt update after adding

```

### 11.3. `firewalld` / `ufw`: Firewall Management

Security is paramount. Ansible must manage host firewalls idempotently.

**Module**

**Purpose**

**Key Parameters**

**`firewalld`**

Manages `firewalld` (common on RHEL/CentOS 7+).

`zone`, `port`, `state`, `permanent`

**`ufw`**

Manages `ufw` (common on Debian/Ubuntu).

`rule`, `port`, `proto`, `state`, `direction`

#### Example: Opening a Production Port Permanently

YAML

```yaml
- name: Open production web server ports on RHEL systems
  hosts: webservers
  when: ansible_os_family == "RedHat"
  tasks:
    - name: Allow HTTPS port 443 permanently
      ansible.posix.firewalld:
        port: 443/tcp
        permanent: yes
        state: enabled
        immediate: yes # Apply immediately without a full reload

    - name: Ensure SSH is allowed in the public zone
      ansible.posix.firewalld:
        service: ssh
        zone: public
        state: enabled
        permanent: yes
        immediate: yes

```

### 11.4. `lookup`: Fetching External Data

The `lookup` plugin allows Ansible to pull data from external sources (files, environment variables, Redis, etc.) _on the control node_ during playbook compilation.2

#### Example: Using `file` Lookup for Certificate Content

If you need to inject the literal content of a certificate or private key into a template, using a `lookup` is much cleaner than loading it into a variable.

YAML

```yaml
- name: Deploy an SSL certificate using file lookup
  hosts: webservers
  vars:
    # This reads the content of the file 'certs/app.crt' from the control node
    ssl_cert_content: "{{ lookup('file', 'certs/app.crt') }}"
    ssl_key_content: "{{ lookup('file', 'certs/app.key') }}"
  tasks:
    - name: Copy SSL certificate content to the managed node
      ansible.builtin.copy:
        content: "{{ ssl_cert_content }}"
        dest: /etc/ssl/certs/app.crt
        owner: root
        group: root
        mode: '0600'
        
    - name: Use the key content in a template (e.g., for NGINX config)
      ansible.builtin.template:
        src: nginx_ssl.conf.j2
        dest: /etc/nginx/conf.d/ssl.conf

```

**Key Distinction:**

-   **`lookup`**: Executes on the **Control Node** to _fetch_ data.3
    
-   **`copy`**: Executes on the **Managed Node** to _transfer_ files.4
    

By mastering these modules, you cover the majority of tasks required in a modern DevOps pipeline, from application delivery and host security to package repository maintenance.

This concludes the expanded **Junior DevOps Ansible Bible**.

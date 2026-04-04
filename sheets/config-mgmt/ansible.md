# Ansible (Configuration Management & Automation)

Agentless automation tool that configures systems over SSH using YAML playbooks.

## Ad-Hoc Commands

### Run a command on all hosts

```bash
ansible all -m shell -a "uptime"
```

### Run on a specific group

```bash
ansible webservers -m ping
```

### Copy a file to all hosts

```bash
ansible all -m copy -a "src=/tmp/hosts dest=/etc/hosts mode=0644" --become
```

### Install a package

```bash
ansible all -m apt -a "name=nginx state=present" --become
```

### Restart a service

```bash
ansible all -m service -a "name=nginx state=restarted" --become
```

### Create a user

```bash
ansible all -m user -a "name=deploy shell=/bin/bash groups=sudo" --become
```

### Manage files and directories

```bash
ansible all -m file -a "path=/opt/app state=directory owner=deploy mode=0755" --become
```

## Inventory

### INI format (/etc/ansible/hosts)

```bash
# Static inventory
[webservers]
web1.example.com
web2.example.com ansible_port=2222

[dbservers]
db1.example.com ansible_user=admin

[production:children]
webservers
dbservers

[webservers:vars]
http_port=80
```

### Dynamic inventory

```bash
ansible-inventory --list -i inventory/aws_ec2.yml
ansible all -i inventory/aws_ec2.yml -m ping
```

## Playbooks

### Run a playbook

```bash
ansible-playbook site.yml
ansible-playbook site.yml --limit webservers
ansible-playbook site.yml --tags "deploy,config"
ansible-playbook site.yml --skip-tags "slow"
ansible-playbook site.yml --check          # dry run
ansible-playbook site.yml --diff           # show file changes
ansible-playbook site.yml -e "version=2.1" # extra vars
```

### Playbook structure

```bash
# site.yml
# ---
# - hosts: webservers
#   become: true
#   vars:
#     app_port: 8080
#   roles:
#     - common
#     - nginx
#   tasks:
#     - name: Ensure app directory exists
#       file:
#         path: /opt/app
#         state: directory
#   handlers:
#     - name: restart nginx
#       service:
#         name: nginx
#         state: restarted
```

## Roles

### Create a role skeleton

```bash
ansible-galaxy role init my_role
```

### Role directory structure

```bash
# roles/my_role/
#   tasks/main.yml       <- entry point
#   handlers/main.yml    <- handlers
#   templates/           <- Jinja2 templates
#   files/               <- static files
#   vars/main.yml        <- role variables (high priority)
#   defaults/main.yml    <- default variables (low priority)
#   meta/main.yml        <- dependencies
```

## Variables & Templates

### Variable precedence (lowest to highest)

```bash
# defaults/main.yml -> group_vars/ -> host_vars/ -> playbook vars
# -> role vars -> extra vars (-e)
```

### Jinja2 template example

```bash
# templates/nginx.conf.j2
# server {
#     listen {{ http_port }};
#     server_name {{ ansible_hostname }};
#     root {{ doc_root }};
# }
```

### Use template module

```bash
# - name: Deploy nginx config
#   template:
#     src: nginx.conf.j2
#     dest: /etc/nginx/sites-available/default
#   notify: restart nginx
```

## Vault

### Create an encrypted file

```bash
ansible-vault create secrets.yml
```

### Encrypt an existing file

```bash
ansible-vault encrypt vars/prod.yml
```

### Edit encrypted file

```bash
ansible-vault edit secrets.yml
```

### Run playbook with vault

```bash
ansible-playbook site.yml --ask-vault-pass
ansible-playbook site.yml --vault-password-file ~/.vault_pass
```

### Encrypt a single string

```bash
ansible-vault encrypt_string 'supersecret' --name 'db_password'
```

## Galaxy

### Install a role from Galaxy

```bash
ansible-galaxy install geerlingguy.docker
```

### Install from requirements file

```bash
ansible-galaxy install -r requirements.yml
```

### Install a collection

```bash
ansible-galaxy collection install community.general
```

## Common Modules

### apt / yum / dnf

```bash
# - apt: name=nginx state=latest update_cache=yes
# - yum: name=httpd state=present
```

### shell vs command

```bash
# command: runs without shell — no pipes, redirects, or env vars
# shell: runs through /bin/sh — supports pipes and redirects
# - shell: cat /etc/passwd | grep deploy
# - command: ls /opt/app
```

### lineinfile

```bash
# - lineinfile:
#     path: /etc/ssh/sshd_config
#     regexp: '^PermitRootLogin'
#     line: 'PermitRootLogin no'
#   notify: restart sshd
```

### cron

```bash
# - cron:
#     name: "backup db"
#     minute: "0"
#     hour: "2"
#     job: "/usr/local/bin/backup.sh"
```

## Tips

- Use `--check --diff` together to preview changes without applying them.
- `ansible-playbook site.yml -vvv` for maximum debug output.
- The `command` module is safer than `shell` — use `shell` only when you need pipes or redirects.
- `become: true` is the modern replacement for the deprecated `sudo: yes`.
- Put secrets in Vault, never in plain vars files committed to version control.
- Use `ansible-lint` to catch anti-patterns before they hit production.
- `gather_facts: false` speeds up playbooks that do not need host facts.
- Tags let you run subsets of a playbook: always tag deploy steps separately from config steps.

## See Also

- terraform
- salt
- puppet
- chef
- ssh
- packer
- vagrant

## References

- [Ansible Documentation](https://docs.ansible.com/ansible/latest/)
- [Ansible Module Index](https://docs.ansible.com/ansible/latest/collections/index_module.html)
- [Ansible Playbook Guide](https://docs.ansible.com/ansible/latest/playbook_guide/index.html)
- [Ansible Inventory Guide](https://docs.ansible.com/ansible/latest/inventory_guide/index.html)
- [Ansible Built-in Modules](https://docs.ansible.com/ansible/latest/collections/ansible/builtin/index.html)
- [Ansible Galaxy](https://galaxy.ansible.com/)
- [Ansible Vault](https://docs.ansible.com/ansible/latest/vault_guide/index.html)
- [Jinja2 Templating in Ansible](https://docs.ansible.com/ansible/latest/playbook_guide/playbooks_templating.html)
- [Ansible GitHub Repository](https://github.com/ansible/ansible)
- [Ansible Best Practices](https://docs.ansible.com/ansible/latest/tips_tricks/ansible_tips_tricks.html)
- [ansible-playbook(1) man page](https://docs.ansible.com/ansible/latest/cli/ansible-playbook.html)

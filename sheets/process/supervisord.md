# supervisord (process manager)

Supervisor is a process control system for Unix that monitors and controls long-running processes, providing automatic restart on failure, log management, process grouping, event listeners for custom monitoring, and a CLI and web interface for runtime management.

## Installation

### Setup Supervisor

```bash
# Install via pip
pip install supervisor

# Install via package manager (Debian/Ubuntu)
sudo apt install supervisor

# Install via package manager (RHEL/Fedora)
sudo dnf install supervisor

# Generate default config
echo_supervisord_conf > /etc/supervisord.conf

# Start supervisord
supervisord -c /etc/supervisord.conf

# Start with nodaemon (foreground, useful in Docker)
supervisord -n -c /etc/supervisord.conf

# Check if running
supervisorctl status
```

## Configuration

### Main Config File

```ini
; /etc/supervisord.conf (or /etc/supervisor/supervisord.conf)

[unix_http_server]
file=/var/run/supervisor.sock
chmod=0700

[inet_http_server]
port=127.0.0.1:9001
username=admin
password=secret

[supervisord]
logfile=/var/log/supervisor/supervisord.log
logfile_maxbytes=50MB
logfile_backups=10
loglevel=info
pidfile=/var/run/supervisord.pid
nodaemon=false
minfds=1024
minprocs=200

[rpcinterface:supervisor]
supervisor.rpcinterface_factory = supervisor.rpcinterface:make_main_rpcinterface

[supervisorctl]
serverurl=unix:///var/run/supervisor.sock

[include]
files = /etc/supervisor/conf.d/*.conf
```

### Program Sections

```ini
; /etc/supervisor/conf.d/myapp.conf

[program:myapp]
command=/usr/bin/python /opt/myapp/app.py --port 8080
directory=/opt/myapp
user=appuser
autostart=true
autorestart=true
startsecs=5
startretries=3
stopwaitsecs=10
stopsignal=TERM
stopasgroup=true
killasgroup=true
redirect_stderr=true
stdout_logfile=/var/log/supervisor/myapp.log
stdout_logfile_maxbytes=50MB
stdout_logfile_backups=5
stderr_logfile=/var/log/supervisor/myapp-error.log
environment=NODE_ENV="production",PORT="8080",SECRET="%(ENV_APP_SECRET)s"
```

### Multiple Instances (numprocs)

```ini
; Run multiple worker instances
[program:worker]
command=/usr/bin/python /opt/myapp/worker.py --id %(process_num)s
process_name=%(program_name)s_%(process_num)02d
numprocs=4
numprocs_start=0
directory=/opt/myapp
user=appuser
autostart=true
autorestart=true
startsecs=3
startretries=5
stdout_logfile=/var/log/supervisor/worker_%(process_num)02d.log
stderr_logfile=/var/log/supervisor/worker_%(process_num)02d-error.log
```

### Restart Behavior

```ini
; autorestart options
[program:critical-service]
command=/usr/bin/my-service
autostart=true

; autorestart=true       -- restart on any exit
; autorestart=false      -- never auto-restart
; autorestart=unexpected -- restart only on unexpected exit codes

autorestart=unexpected
exitcodes=0              ; expected exit codes (default: 0)
startsecs=10             ; must run this long to be "started" (default: 1)
startretries=5           ; max retry attempts (default: 3)

; Backoff: supervisor waits between retries
; Retry 1: immediate, Retry 2: 1s, Retry 3: 2s, etc.
```

## Process Groups

### Group Management

```ini
; Group related processes
[group:webstack]
programs=nginx-proxy,myapp,worker
priority=999

[program:nginx-proxy]
command=/usr/sbin/nginx -g "daemon off;"
priority=100
autostart=true
autorestart=true

[program:myapp]
command=/opt/myapp/bin/server
priority=200
autostart=true
autorestart=true

[program:worker]
command=/opt/myapp/bin/worker
priority=300
autostart=true
autorestart=true
```

```bash
# Control entire group
supervisorctl start webstack:*
supervisorctl stop webstack:*
supervisorctl restart webstack:*

# Control individual member
supervisorctl start webstack:myapp
supervisorctl stop webstack:worker
```

## supervisorctl Commands

### Runtime Management

```bash
# Status of all processes
supervisorctl status

# Start/stop/restart a process
supervisorctl start myapp
supervisorctl stop myapp
supervisorctl restart myapp

# Start/stop all processes
supervisorctl start all
supervisorctl stop all

# Reload config (add/remove changed programs)
supervisorctl reread
supervisorctl update

# Reread and apply in one step
supervisorctl reread && supervisorctl update

# Tail process logs
supervisorctl tail myapp
supervisorctl tail -f myapp           # follow mode
supervisorctl tail myapp stderr       # stderr log

# Clear process logs
supervisorctl clear myapp
supervisorctl clear all

# Get process info
supervisorctl pid myapp
supervisorctl pid all

# Send signal to process
supervisorctl signal SIGHUP myapp
supervisorctl signal SIGUSR1 worker:worker_00

# Interactive mode
supervisorctl
# supervisor> status
# supervisor> restart myapp
# supervisor> quit

# Shutdown supervisord entirely
supervisorctl shutdown
```

## Event Listeners

### Custom Event Handling

```ini
; Event listener configuration
[eventlistener:crashmail]
command=/usr/bin/python /opt/scripts/crashmail.py
events=PROCESS_STATE_EXITED
buffer_size=10
```

```python
#!/usr/bin/env python
# /opt/scripts/crashmail.py -- email on process crash
import sys
import subprocess

def write_stdout(s):
    sys.stdout.write(s)
    sys.stdout.flush()

def write_stderr(s):
    sys.stderr.write(s)
    sys.stderr.flush()

def main():
    while True:
        write_stdout('READY\n')  # signal ready for event
        header = sys.stdin.readline()
        headers = dict(x.split(':') for x in header.split())
        data = sys.stdin.read(int(headers['len']))
        payload = dict(x.split(':') for x in data.split())

        if headers['eventname'] == 'PROCESS_STATE_EXITED':
            if int(payload.get('expected', '1')) == 0:
                name = payload['processname']
                write_stderr(f'UNEXPECTED EXIT: {name}\n')
                subprocess.run([
                    'mail', '-s', f'Process {name} crashed',
                    'ops@example.com'
                ], input=f'Process {name} exited unexpectedly'.encode())

        write_stdout('RESULT 2\nOK')  # event processed

if __name__ == '__main__':
    main()
```

### Available Events

```bash
# Process state events
# PROCESS_STATE_STARTING    -- process is starting
# PROCESS_STATE_RUNNING     -- process entered running state
# PROCESS_STATE_BACKOFF     -- process failed to start, retrying
# PROCESS_STATE_STOPPING    -- process is stopping
# PROCESS_STATE_EXITED      -- process exited (check expected flag)
# PROCESS_STATE_STOPPED     -- process stopped by user
# PROCESS_STATE_FATAL       -- process could not start after retries

# Supervisor events
# SUPERVISOR_STATE_CHANGE_RUNNING
# SUPERVISOR_STATE_CHANGE_STOPPING

# Tick events (heartbeat)
# TICK_5, TICK_60, TICK_3600
```

## Docker Integration

### Supervisor in Containers

```dockerfile
FROM python:3.12-slim
RUN pip install supervisor
COPY supervisord.conf /etc/supervisord.conf
COPY conf.d/ /etc/supervisor/conf.d/
CMD ["supervisord", "-n", "-c", "/etc/supervisord.conf"]
```

```ini
; Docker-friendly supervisord.conf
[supervisord]
nodaemon=true
logfile=/dev/stdout
logfile_maxbytes=0
loglevel=info

[program:app]
command=python /app/server.py
stdout_logfile=/dev/stdout
stdout_logfile_maxbytes=0
stderr_logfile=/dev/stderr
stderr_logfile_maxbytes=0
autorestart=true
```

## Tips

- Set `stopasgroup=true` and `killasgroup=true` to ensure child processes are killed when stopping a program
- Use `startsecs=5` (or higher) to prevent rapid restart loops for processes that crash immediately after starting
- Set `redirect_stderr=true` to merge stdout and stderr into a single log file for simpler log management
- Use `%(ENV_VAR)s` syntax in config to reference host environment variables without hardcoding secrets
- Always run `reread` then `update` after config changes instead of restarting supervisord entirely
- Use `numprocs` with `%(process_num)s` to run multiple identical workers with unique identifiers and log files
- Event listeners enable custom monitoring (email, Slack, PagerDuty) without polling or external cron jobs
- Set `nodaemon=true` when running in Docker so the container stays in the foreground
- Use `priority` values in program sections to control startup order within a group (lower = starts first)
- Log rotation via `stdout_logfile_maxbytes` and `stdout_logfile_backups` prevents disk exhaustion

## See Also

- systemd, cron, pm2, monit, runit, s6

## References

- [Supervisor Documentation](http://supervisord.org/)
- [Supervisor Configuration](http://supervisord.org/configuration.html)
- [Supervisor Events](http://supervisord.org/events.html)
- [Supervisor XML-RPC API](http://supervisord.org/api.html)
- [Supervisor GitHub](https://github.com/Supervisor/supervisor)

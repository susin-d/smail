# Memory Optimization Strategies

## Current Memory Budget

| Service    | Limit  | Key Tuning                                       |
|------------|--------|--------------------------------------------------|
| Postfix    | 100 MB | `default_process_limit=10`, connection throttling |
| Dovecot    | 150 MB | `vsz_limit=64M` login, `128M` imap process       |
| MariaDB    | 80 MB  | `innodb_buffer_pool_size=32M`, `performance_schema=OFF` |
| Redis      | 50 MB  | `maxmemory 40mb`, `allkeys-lru`, persistence off  |
| Go API     | 150 MB | single static binary, low-overhead goroutines     |
| OpenDKIM   | 50 MB  | Minimal footprint, single process                 |
| Nginx      | 30 MB  | 1 worker, 256 connections                         |
| **Total**  | **610 MB** | **~400 MB headroom for OS**                  |

## Per-Service Optimization Details

### MariaDB

These settings in `my.cnf` reduce MariaDB from its default ~400 MB to fit in 80 MB:

```ini
innodb_buffer_pool_size = 32M    # Default 128M → 32M
key_buffer_size = 8M             # Default 16M → 8M
max_connections = 20             # Default 151 → 20
innodb_log_file_size = 16M       # Default 48M → 16M
performance_schema = OFF         # Saves ~20 MB
table_open_cache = 64            # Default 2000 → 64
sort_buffer_size = 256K          # Default 2M → 256K
read_buffer_size = 256K          # Default 128K → 256K
query_cache_size = 0             # Disabled (unnecessary with small dataset)
```

### Redis

Redis is configured for minimal memory without persistence:

```
maxmemory 40mb                   # Hard cap
maxmemory-policy allkeys-lru     # Evict least-recently-used
save ""                          # No RDB snapshots
appendonly no                    # No AOF log
```

### Postfix

```
default_process_limit = 10       # Max 10 simultaneous processes
smtpd_client_connection_count_limit = 5
smtpd_client_connection_rate_limit = 10
```

### Dovecot

```
service imap-login {
  process_min_avail = 1          # Only 1 pre-forked login process
  vsz_limit = 64M               # Memory limit per login process
}
service imap {
  vsz_limit = 128M              # Memory limit per IMAP connection
}
mail_max_userip_connections = 10 # Limit connections per user
```

### Go API

- **Single process binary** keeps baseline memory lower than multi-worker Python setups
- **Goroutine concurrency** handles high I/O parallelism without extra worker processes
- **DB connection pooling** is configured in the Go service to cap memory usage
- Prefer low container memory limits (current `150m`) to prevent runaway usage

### Nginx

```
worker_processes 1;              # Single worker
worker_connections 256;          # Low connection limit
keepalive_timeout 30;            # Short keepalive
```

## Monitoring Memory

```bash
# Real-time container memory usage
docker stats

# One-shot view
docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}\t{{.MemPerc}}"

# System total
free -h

# Per-process inside a container
docker exec smail-mariadb cat /proc/meminfo | head -5
```

## Emergency Memory Reduction

If memory exceeds limits:

1. **Reduce MariaDB buffer pool** to 16M: `SET GLOBAL innodb_buffer_pool_size=16777216;`
2. **Reduce Redis maxmemory** to 20MB: `redis-cli CONFIG SET maxmemory 20mb`
3. **Lower Go API container limit** (for emergency): edit `docker-compose.yml` `api-go.mem_limit`
4. **Disable IMAP idle** connections
5. **Lower Postfix process limits** further

## Swap Configuration (Safety Net)

Add a small swap file for emergency overflow:

```bash
fallocate -l 512M /swapfile
chmod 600 /swapfile
mkswap /swapfile
swapon /swapfile
echo '/swapfile none swap sw 0 0' >> /etc/fstab

# Reduce swappiness
echo 'vm.swappiness=10' >> /etc/sysctl.conf
sysctl -p
```

## Lightweight Monitoring Suggestions

Avoid Prometheus/Grafana (too heavy). Instead use:

1. **ctop** — Container top (single binary, <5 MB)
   ```bash
   wget https://github.com/bcicen/ctop/releases/download/v0.7.7/ctop-0.7.7-linux-amd64 -O /usr/local/bin/ctop
   chmod +x /usr/local/bin/ctop
   ctop
   ```

2. **Docker healthchecks** — Already configured in docker-compose.yml

3. **Cron-based alerts** — Simple script to check memory:
   ```bash
   #!/bin/bash
   USED=$(free -m | awk 'NR==2{print $3}')
   if [ "$USED" -gt 900 ]; then
    echo "HIGH MEMORY: ${USED}MB" | mail -s "smail memory alert" admin@yourdomain.com
   fi
   ```
  Add to crontab: `*/5 * * * * /opt/smail/check_memory.sh`

4. **Log rotation** — Prevent log files from consuming disk:
   ```bash
   # /etc/logrotate.d/docker
   /var/lib/docker/containers/*/*.log {
     rotate 3
     daily
     compress
     size 10M
     missingok
   }
   ```

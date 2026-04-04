# Celery (Distributed Task Queue)

Celery is a distributed task queue for Python that processes millions of tasks per day, supporting real-time processing and task scheduling with pluggable brokers (RabbitMQ, Redis), result backends, and multiple concurrency models.

## Worker Management
### Starting Workers
```bash
# Basic worker
celery -A myapp worker --loglevel=info

# Named worker with concurrency
celery -A myapp worker --loglevel=info \
    --hostname=worker1@%h \
    --concurrency=8 \
    --pool=prefork

# Eventlet for I/O-bound tasks
celery -A myapp worker --pool=eventlet --concurrency=500

# Gevent pool
celery -A myapp worker --pool=gevent --concurrency=200

# Consume from specific queues
celery -A myapp worker --queues=high,default,low

# Autoscale between min and max processes
celery -A myapp worker --autoscale=10,3
```

### Worker Inspection
```bash
# List active workers
celery -A myapp inspect active
celery -A myapp inspect reserved
celery -A myapp inspect scheduled

# Worker stats
celery -A myapp inspect stats

# Ping all workers
celery -A myapp inspect ping

# Revoke a task
celery -A myapp control revoke <task-id> --terminate

# Rate limit a task type
celery -A myapp control rate_limit myapp.tasks.send_email 10/m

# Shutdown a worker gracefully
celery -A myapp control shutdown --destination worker1@hostname
```

## Task Definition
### Basic Tasks
```python
from celery import Celery

app = Celery('myapp', broker='redis://localhost:6379/0',
             backend='redis://localhost:6379/1')

@app.task(bind=True, max_retries=3, default_retry_delay=60)
def process_order(self, order_id):
    try:
        order = Order.objects.get(id=order_id)
        order.process()
        return {'status': 'completed', 'order_id': order_id}
    except TemporaryError as exc:
        raise self.retry(exc=exc, countdown=2 ** self.request.retries)

@app.task(ignore_result=True, rate_limit='100/m')
def send_notification(user_id, message):
    """Fire-and-forget task with rate limiting."""
    send_push(user_id, message)

@app.task(time_limit=300, soft_time_limit=240)
def heavy_computation(data):
    """Task with hard and soft time limits."""
    return compute(data)
```

### Task Options
```python
@app.task(
    name='myapp.tasks.critical_job',
    bind=True,
    max_retries=5,
    default_retry_delay=30,
    rate_limit='50/h',
    time_limit=600,
    soft_time_limit=540,
    acks_late=True,
    reject_on_worker_lost=True,
    ignore_result=False,
    serializer='json',
    queue='critical',
    priority=9,
)
def critical_job(self, payload):
    pass
```

## Beat Scheduler
```python
# celeryconfig.py
from celery.schedules import crontab

beat_schedule = {
    'daily-report': {
        'task': 'myapp.tasks.generate_report',
        'schedule': crontab(hour=6, minute=0),
        'args': ('daily',),
    },
    'cleanup-every-hour': {
        'task': 'myapp.tasks.cleanup_stale',
        'schedule': 3600.0,  # seconds
    },
    'weekly-digest': {
        'task': 'myapp.tasks.send_digest',
        'schedule': crontab(hour=9, minute=0, day_of_week='monday'),
        'kwargs': {'digest_type': 'weekly'},
    },
}
```

```bash
# Start beat scheduler
celery -A myapp beat --loglevel=info

# Beat with database scheduler (django-celery-beat)
celery -A myapp beat --scheduler django_celery_beat.schedulers:DatabaseScheduler

# Embedded beat (dev only -- runs beat inside worker)
celery -A myapp worker --beat --loglevel=info
```

## Result Backends
```python
# Redis backend
app.conf.result_backend = 'redis://localhost:6379/1'
app.conf.result_expires = 3600  # 1 hour

# RabbitMQ RPC backend
app.conf.result_backend = 'rpc://'

# Database backend (SQLAlchemy)
app.conf.result_backend = 'db+postgresql://user:pass@localhost/celery_results'

# Retrieve results
result = process_order.delay(order_id=42)
result.ready()      # True/False
result.successful()  # True/False
result.get(timeout=10)
result.state         # PENDING, STARTED, SUCCESS, FAILURE, RETRY
result.traceback     # on failure
```

## Routing & Queues
```python
# Route tasks to specific queues
app.conf.task_routes = {
    'myapp.tasks.send_email': {'queue': 'email'},
    'myapp.tasks.process_image': {'queue': 'media', 'routing_key': 'media.image'},
    'myapp.tasks.*': {'queue': 'default'},
}

# Priority queues (Redis)
app.conf.broker_transport_options = {
    'priority_steps': list(range(10)),
    'sep': ':',
    'queue_order_strategy': 'priority',
}

# Manual routing on call
process_order.apply_async(args=[42], queue='high_priority', priority=9)
```

## Chains, Chords & Groups
```python
from celery import chain, group, chord, signature

# Chain: sequential execution, output pipes to next
pipeline = chain(
    extract.s(source='api'),
    transform.s(),
    load.s(target='warehouse'),
)
result = pipeline.apply_async()

# Group: parallel execution
batch = group(
    process_item.s(item_id) for item_id in range(100)
)
result = batch.apply_async()

# Chord: group + callback when all complete
workflow = chord(
    [fetch_page.s(url) for url in urls],
    aggregate_results.s()
)
result = workflow.apply_async()

# Complex workflow composition
workflow = chain(
    extract.s(),
    chord(
        group(transform.s(shard) for shard in range(4)),
        merge.s(),
    ),
    load.s(),
)
```

## Rate Limiting
```python
# Per-task decorator
@app.task(rate_limit='10/s')    # 10 per second
@app.task(rate_limit='100/m')   # 100 per minute
@app.task(rate_limit='1000/h')  # 1000 per hour

# Dynamic rate limiting at runtime
celery -A myapp control rate_limit myapp.tasks.send_email 50/m

# Token bucket via custom base task
from celery import Task
import redis

class RateLimitedTask(Task):
    _bucket = None

    @property
    def bucket(self):
        if self._bucket is None:
            self._bucket = redis.Redis()
        return self._bucket

    def __call__(self, *args, **kwargs):
        key = f'ratelimit:{self.name}'
        if self.bucket.incr(key) == 1:
            self.bucket.expire(key, 60)
        if int(self.bucket.get(key)) > 100:
            self.retry(countdown=60)
        return super().__call__(*args, **kwargs)
```

## Monitoring (Flower)
```bash
# Install and start Flower
pip install flower
celery -A myapp flower --port=5555

# With authentication
celery -A myapp flower --basic_auth=admin:secret

# Prometheus metrics
celery -A myapp flower --enable_events

# CLI events monitor
celery -A myapp events --dump

# Purge all messages from broker
celery -A myapp purge
```

## Configuration
```python
# celeryconfig.py
broker_url = 'amqp://guest:guest@localhost:5672//'
result_backend = 'redis://localhost:6379/1'

task_serializer = 'json'
result_serializer = 'json'
accept_content = ['json']
timezone = 'UTC'
enable_utc = True

worker_prefetch_multiplier = 1     # fair scheduling
worker_max_tasks_per_child = 1000  # recycle after N tasks
worker_max_memory_per_child = 200000  # 200MB limit

task_acks_late = True
task_reject_on_worker_lost = True
task_track_started = True

broker_connection_retry_on_startup = True
broker_pool_limit = 10
```

## Tips
- Set `task_acks_late=True` and `reject_on_worker_lost=True` for at-least-once delivery
- Use `ignore_result=True` on fire-and-forget tasks to avoid filling the result backend
- Set `worker_prefetch_multiplier=1` for fair task distribution across workers
- Recycle workers with `max_tasks_per_child` to prevent memory leaks in long-running processes
- Use `soft_time_limit` to handle timeouts gracefully before the hard `time_limit` kills the task
- Prefer Redis as broker for simplicity; use RabbitMQ when you need advanced routing or durability
- Always set `result_expires` to prevent result backend from growing unbounded
- Use chords sparingly -- they add complexity and a chord unlock overhead per group
- Monitor with Flower in production and set up alerting on queue depth
- Bind tasks (`bind=True`) to access `self.retry()` and `self.request` metadata
- Use JSON serializer in production -- pickle is a security risk with untrusted inputs
- Test tasks synchronously with `CELERY_ALWAYS_EAGER=True` in unit tests

## See Also
- redis, rabbitmq, airflow, sidekiq, bull

## References
- [Celery Documentation](https://docs.celeryq.dev/en/stable/)
- [Celery Best Practices](https://docs.celeryq.dev/en/stable/userguide/tasks.html#best-practices)
- [Flower Monitoring](https://flower.readthedocs.io/en/latest/)
- [django-celery-beat](https://django-celery-beat.readthedocs.io/en/latest/)
- [Celery GitHub Repository](https://github.com/celery/celery)

# Sidekiq (Background Job Processing for Ruby)

Sidekiq is a high-performance, Redis-backed background job processor for Ruby that uses threads for concurrency, providing efficient memory usage and simple deployment for millions of jobs per day.

## Worker Definition
### Basic Workers
```ruby
# app/workers/order_processor.rb
class OrderProcessor
  include Sidekiq::Job

  sidekiq_options queue: 'critical',
                  retry: 5,
                  backtrace: true,
                  lock: :until_executed

  def perform(order_id)
    order = Order.find(order_id)
    order.process!
    NotificationWorker.perform_async(order.user_id, 'Order processed')
  end
end

# Enqueue a job
OrderProcessor.perform_async(42)
OrderProcessor.perform_in(5.minutes, 42)
OrderProcessor.perform_at(Time.now + 1.hour, 42)
```

### Job Options
```ruby
class HeavyWorker
  include Sidekiq::Job

  sidekiq_options(
    queue: 'low',
    retry: 10,
    dead: true,              # send to dead queue on exhaust
    backtrace: 20,           # lines of backtrace to store
    unique_for: 1.hour,      # deduplicate (Enterprise)
    tags: ['reports'],
    lock: :until_executed,   # Enterprise lock
  )

  sidekiq_retry_in do |count, exception|
    # Custom retry delay: exponential with jitter
    (count ** 4) + 15 + (rand(10) * (count + 1))
  end

  sidekiq_retries_exhausted do |msg, exception|
    Sentry.capture_exception(exception)
    DeadJobNotifier.notify(msg)
  end

  def perform(report_id)
    report = Report.generate(report_id)
    ReportMailer.deliver(report)
  end
end
```

## Configuration
### sidekiq.yml
```yaml
# config/sidekiq.yml
:concurrency: 25
:timeout: 25
:max_retries: 25

:queues:
  - [critical, 6]
  - [default, 4]
  - [low, 2]
  - [bulk, 1]

# Strict ordering (process critical first)
# :queues:
#   - critical
#   - default
#   - low
```

### Initializer
```ruby
# config/initializers/sidekiq.rb
Sidekiq.configure_server do |config|
  config.redis = { url: ENV['REDIS_URL'], pool_size: 25 }
  config.on(:startup) { puts "Sidekiq started" }
  config.on(:shutdown) { puts "Sidekiq shutting down" }

  config.death_handlers << ->(job, exception) do
    SlackNotifier.alert("Job #{job['class']} died: #{exception.message}")
  end
end

Sidekiq.configure_client do |config|
  config.redis = { url: ENV['REDIS_URL'], pool_size: 5 }
end
```

## Process Management
```bash
# Start Sidekiq
bundle exec sidekiq -C config/sidekiq.yml

# Specify queues on command line
bundle exec sidekiq -q critical,6 -q default,4 -q low,2

# Set concurrency
bundle exec sidekiq -c 25

# Require specific file
bundle exec sidekiq -r ./app/workers/boot.rb

# Quiet (stop fetching new jobs, finish current)
kill -TSTP $(cat tmp/pids/sidekiq.pid)

# Terminate
kill -TERM $(cat tmp/pids/sidekiq.pid)

# systemd service
# /etc/systemd/system/sidekiq.service
# [Service]
# Type=notify
# ExecStart=/usr/bin/bundle exec sidekiq -C config/sidekiq.yml
# WatchdogSec=10
# Restart=on-failure
```

## Middleware
### Server Middleware
```ruby
class LoggingMiddleware
  def call(worker, job, queue)
    start = Process.clock_gettime(Process::CLOCK_MONOTONIC)
    yield
  ensure
    elapsed = Process.clock_gettime(Process::CLOCK_MONOTONIC) - start
    Rails.logger.info("[Sidekiq] #{worker.class} #{elapsed.round(3)}s")
  end
end

Sidekiq.configure_server do |config|
  config.server_middleware do |chain|
    chain.add LoggingMiddleware
  end
end
```

### Client Middleware
```ruby
class UniqueJobMiddleware
  def call(worker_class, job, queue, redis_pool)
    key = "unique:#{worker_class}:#{Digest::SHA256.hexdigest(job['args'].to_json)}"
    return false if Sidekiq.redis { |c| !c.set(key, 1, nx: true, ex: 3600) }
    yield
  end
end

Sidekiq.configure_client do |config|
  config.client_middleware do |chain|
    chain.add UniqueJobMiddleware
  end
end
```

## Retry System
```ruby
# Default retry schedule (exponential backoff with jitter):
# retry_count ** 4 + 15 + (rand(10) * (retry_count + 1))
#
# Retry 1:  ~16-25 seconds
# Retry 2:  ~31-55 seconds
# Retry 3:  ~96-126 seconds (~2 min)
# Retry 10: ~10015-10125 seconds (~2.8 hours)
# Retry 25: ~390640-390890 seconds (~4.5 days)

# Skip retries for specific errors
class PaymentWorker
  include Sidekiq::Job
  sidekiq_options retry: 5

  sidekiq_retry_in do |count, exception|
    case exception
    when InvalidCardError
      :kill  # send directly to dead queue
    when RateLimitError
      60     # fixed 1 minute delay
    else
      (count ** 4) + 15
    end
  end

  def perform(payment_id)
    PaymentService.charge(payment_id)
  end
end
```

## Scheduled Jobs
```ruby
# Perform later
MyWorker.perform_in(30.minutes, args)
MyWorker.perform_at(Date.tomorrow.noon, args)

# Periodic jobs (Enterprise)
Sidekiq.configure_server do |config|
  config.periodic do |mgr|
    mgr.register('0 6 * * *', ReportWorker, retry: 3)
    mgr.register('*/5 * * * *', HealthCheckWorker)
    mgr.register('0 0 1 * *', MonthlyCleanupWorker, queue: 'low')
  end
end

# Using sidekiq-cron gem
Sidekiq::Cron::Job.create(
  name: 'Daily cleanup',
  cron: '0 3 * * *',
  class: 'CleanupWorker',
  queue: 'low',
)
```

## Batches (Pro/Enterprise)
```ruby
# Sidekiq Pro Batches
batch = Sidekiq::Batch.new
batch.description = "Import 10k records"
batch.on(:complete, ImportCallback, 'import_id' => 42)
batch.on(:success, ImportCallback)

batch.jobs do
  1000.times do |i|
    ImportRowWorker.perform_async(i)
  end
end

class ImportCallback
  def on_complete(status, options)
    puts "Batch #{status.bid} complete: #{status.total} jobs, #{status.failures} failures"
  end

  def on_success(status, options)
    Import.find(options['import_id']).mark_complete!
  end
end

# Nested batches
parent = Sidekiq::Batch.new
parent.jobs do
  child = Sidekiq::Batch.new
  child.jobs do
    ChildWorker.perform_async(data)
  end
end
```

## Death Queue & Error Handling
```ruby
# Access dead jobs via API
dead = Sidekiq::DeadSet.new
dead.size
dead.each { |job| puts job.item }

# Retry a dead job
dead.find_job(jid).retry

# Clear dead queue
dead.clear

# RetrySet operations
retries = Sidekiq::RetrySet.new
retries.size
retries.clear
retries.select { |j| j.item['class'] == 'FailingWorker' }.each(&:delete)

# ScheduledSet operations
scheduled = Sidekiq::ScheduledSet.new
scheduled.size
```

## Monitoring (Web UI)
```ruby
# config/routes.rb (Rails)
require 'sidekiq/web'

mount Sidekiq::Web => '/sidekiq'

# With authentication
Sidekiq::Web.use Rack::Auth::Basic do |username, password|
  ActiveSupport::SecurityUtils.secure_compare(
    Digest::SHA256.hexdigest(username), Digest::SHA256.hexdigest(ENV['SIDEKIQ_USER'])
  ) & ActiveSupport::SecurityUtils.secure_compare(
    Digest::SHA256.hexdigest(password), Digest::SHA256.hexdigest(ENV['SIDEKIQ_PASS'])
  )
end
```

```ruby
# API stats
stats = Sidekiq::Stats.new
stats.processed          # total processed
stats.failed             # total failed
stats.enqueued           # currently enqueued
stats.queues             # { "default" => 5, "critical" => 2 }
stats.workers_size       # busy workers

# Queue operations
queue = Sidekiq::Queue.new('default')
queue.size
queue.latency            # seconds oldest job has been waiting
queue.clear
```

## Tips
- Set concurrency to match your database connection pool size (e.g., `pool: 25` in database.yml)
- Use weighted queue priorities (`[critical, 6]`) rather than strict ordering for fairness
- Always handle `Sidekiq::Shutdown` in long-running jobs to checkpoint on graceful shutdown
- Use `perform_bulk` for enqueueing thousands of jobs efficiently in a single Redis round-trip
- Keep job arguments small and serializable -- pass IDs, not full objects
- Set `backtrace: true` in development but limit to `backtrace: 20` in production
- Use death handlers to alert on permanently failed jobs before they pile up
- Redis `maxmemory-policy` must be `noeviction` -- Sidekiq data must never be silently dropped
- Monitor queue latency, not just queue size -- latency reveals actual user impact
- Use `TSTP` signal for graceful quiet (stop fetching) before `TERM` during deploys
- Profile memory with `derailed_benchmarks` gem -- Sidekiq workers share memory via fork
- Sidekiq Pro/Enterprise features (batches, rate limiting, unique jobs) are worth it at scale

## See Also
- redis, celery, bull, resque, activejob

## References
- [Sidekiq Wiki](https://github.com/sidekiq/sidekiq/wiki)
- [Sidekiq Best Practices](https://github.com/sidekiq/sidekiq/wiki/Best-Practices)
- [Sidekiq Pro](https://sidekiq.org/products/pro.html)
- [Sidekiq Enterprise](https://sidekiq.org/products/enterprise.html)
- [Sidekiq in Practice (Nate Berkopec)](https://nateberkopec.com/blog/2021-01-04-sidekiq-in-practice.html)

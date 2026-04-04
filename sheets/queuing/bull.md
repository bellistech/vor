# Bull / BullMQ (Node.js Job Queue)

Bull and BullMQ are Redis-backed job queue libraries for Node.js, providing delayed jobs, rate limiting, repeatable tasks, sandboxed processors, parent-child job flows, and real-time events for building robust distributed task processing systems.

## Queue Setup
### BullMQ (Recommended)
```typescript
import { Queue, Worker, QueueScheduler } from 'bullmq';
import IORedis from 'ioredis';

const connection = new IORedis({
  host: 'localhost',
  port: 6379,
  maxRetriesPerRequest: null,
});

// Create a queue
const emailQueue = new Queue('email', { connection });

// Add a job
await emailQueue.add('welcome', {
  to: 'user@example.com',
  subject: 'Welcome!',
  body: 'Thanks for signing up.',
}, {
  attempts: 3,
  backoff: { type: 'exponential', delay: 1000 },
  removeOnComplete: { count: 1000 },
  removeOnFail: { age: 7 * 24 * 3600 },
});
```

### Bull (Legacy)
```javascript
const Bull = require('bull');

const queue = new Bull('video-transcoding', {
  redis: { host: '127.0.0.1', port: 6379 },
  defaultJobOptions: {
    attempts: 3,
    backoff: { type: 'exponential', delay: 2000 },
    removeOnComplete: true,
  },
});
```

## Workers and Processors
### Basic Worker
```typescript
import { Worker, Job } from 'bullmq';

const worker = new Worker('email', async (job: Job) => {
  console.log(`Processing ${job.name} [${job.id}]`);

  await sendEmail(job.data.to, job.data.subject, job.data.body);

  // Update progress
  await job.updateProgress(50);

  await logDelivery(job.data.to);
  await job.updateProgress(100);

  return { delivered: true, timestamp: Date.now() };
}, {
  connection,
  concurrency: 5,
  limiter: { max: 100, duration: 60000 },  // 100 jobs per minute
});

worker.on('completed', (job, result) => {
  console.log(`Job ${job.id} completed:`, result);
});

worker.on('failed', (job, err) => {
  console.error(`Job ${job?.id} failed:`, err.message);
});
```

### Sandboxed Processors
```typescript
// processor.ts (separate file -- runs in child process)
import { SandboxedJob } from 'bullmq';

export default async function (job: SandboxedJob) {
  // Heavy computation isolated in its own process
  const result = await heavyComputation(job.data);
  return result;
}

// main.ts
const worker = new Worker('heavy', './processor.ts', {
  connection,
  concurrency: 4,         // 4 child processes
  useWorkerThreads: true,  // use worker threads instead of child processes
});
```

## Delayed Jobs
```typescript
// Delay by milliseconds
await queue.add('reminder', { userId: 42 }, {
  delay: 30 * 60 * 1000,  // 30 minutes
});

// Specific timestamp
const delay = new Date('2024-12-25T00:00:00Z').getTime() - Date.now();
await queue.add('christmas-greeting', { message: 'Ho ho ho' }, {
  delay,
});

// Check delayed job count
const delayed = await queue.getDelayedCount();
console.log(`${delayed} jobs waiting`);
```

## Rate Limiting
```typescript
// Worker-level rate limiting
const worker = new Worker('api-calls', processor, {
  connection,
  limiter: {
    max: 10,           // max 10 jobs
    duration: 1000,    // per 1 second
  },
});

// Group-based rate limiting (BullMQ Pro)
await queue.add('api-call', data, {
  group: {
    id: tenantId,
    limit: { max: 5, duration: 60000 },  // 5 per minute per tenant
  },
});

// Queue-level rate limit via job options
await queue.add('webhook', payload, {
  rateLimiterKey: 'webhook-global',
});
```

## Repeatable Jobs
```typescript
// Cron-based repeatable job
await queue.add('daily-report', { type: 'daily' }, {
  repeat: {
    pattern: '0 9 * * *',         // every day at 9am
    tz: 'America/New_York',
  },
});

// Fixed interval
await queue.add('health-check', {}, {
  repeat: {
    every: 30000,                  // every 30 seconds
  },
});

// Limited repeats
await queue.add('trial-reminder', { userId: 42 }, {
  repeat: {
    pattern: '0 10 * * *',
    limit: 7,                      // repeat 7 times then stop
    endDate: new Date('2024-12-31'),
  },
});

// List repeatable jobs
const repeatableJobs = await queue.getRepeatableJobs();
console.log(repeatableJobs);

// Remove a repeatable job
await queue.removeRepeatableByKey(repeatableJobs[0].key);
```

## Flows (Parent-Child Jobs)
```typescript
import { FlowProducer } from 'bullmq';

const flowProducer = new FlowProducer({ connection });

// Define a flow tree
const flow = await flowProducer.add({
  name: 'deploy',
  queueName: 'deployment',
  data: { version: '2.0.0' },
  children: [
    {
      name: 'build',
      queueName: 'ci',
      data: { repo: 'myapp' },
      children: [
        { name: 'test', queueName: 'ci', data: { suite: 'unit' } },
        { name: 'test', queueName: 'ci', data: { suite: 'integration' } },
        { name: 'lint', queueName: 'ci', data: { strict: true } },
      ],
    },
    {
      name: 'migrate-db',
      queueName: 'database',
      data: { version: '2.0.0' },
    },
  ],
});

// Parent waits for all children to complete
// Access parent from child:
const worker = new Worker('ci', async (job) => {
  const result = await runTests(job.data);
  return result;
}, { connection });

// Access children results from parent:
const deployWorker = new Worker('deployment', async (job) => {
  const childValues = await job.getChildrenValues();
  console.log('All dependencies complete:', childValues);
  await deploy(job.data.version);
}, { connection });
```

## Events
```typescript
import { QueueEvents } from 'bullmq';

const queueEvents = new QueueEvents('email', { connection });

queueEvents.on('completed', ({ jobId, returnvalue }) => {
  console.log(`Job ${jobId} completed with:`, returnvalue);
});

queueEvents.on('failed', ({ jobId, failedReason }) => {
  console.error(`Job ${jobId} failed:`, failedReason);
});

queueEvents.on('progress', ({ jobId, data }) => {
  console.log(`Job ${jobId} progress:`, data);
});

queueEvents.on('waiting', ({ jobId }) => {
  console.log(`Job ${jobId} is waiting`);
});

queueEvents.on('delayed', ({ jobId, delay }) => {
  console.log(`Job ${jobId} delayed by ${delay}ms`);
});

// Worker events
worker.on('error', (err) => console.error('Worker error:', err));
worker.on('drained', () => console.log('No more jobs'));
```

## Job Lifecycle & Management
```typescript
// Get a specific job
const job = await queue.getJob(jobId);
console.log(job.data, job.returnvalue, job.failedReason);

// Get jobs by state
const waiting = await queue.getWaiting(0, 100);
const active = await queue.getActive(0, 100);
const failed = await queue.getFailed(0, 100);

// Queue counts
const counts = await queue.getJobCounts('waiting', 'active', 'completed', 'failed', 'delayed');

// Retry all failed jobs
for (const j of await queue.getFailed()) { await j.retry(); }

// Drain (remove waiting), obliterate (remove all), pause/resume
await queue.drain();
await queue.obliterate({ force: true });
await queue.pause();
await queue.resume();
```

## Dashboard (Bull Board)
```typescript
import { createBullBoard } from '@bull-board/api';
import { BullMQAdapter } from '@bull-board/api/bullMQAdapter';
import { ExpressAdapter } from '@bull-board/express';
import express from 'express';

const serverAdapter = new ExpressAdapter();
serverAdapter.setBasePath('/admin/queues');

createBullBoard({
  queues: [
    new BullMQAdapter(emailQueue),
    new BullMQAdapter(videoQueue),
    new BullMQAdapter(reportQueue),
  ],
  serverAdapter,
});

const app = express();
app.use('/admin/queues', serverAdapter.getRouter());
app.listen(3000);
```

## Configuration
```typescript
// Queue options
const queue = new Queue('myqueue', {
  connection,
  prefix: 'myapp',                      // Redis key prefix
  defaultJobOptions: {
    attempts: 3,
    backoff: { type: 'exponential', delay: 1000 },
    removeOnComplete: { count: 5000 },   // keep last 5000
    removeOnFail: { age: 30 * 24 * 3600 }, // keep 30 days
  },
});

// Worker options
const worker = new Worker('myqueue', processor, {
  connection,
  concurrency: 10,
  lockDuration: 30000,                   // 30s lock per job
  lockRenewTime: 15000,                  // renew lock every 15s
  stalledInterval: 30000,                // check stalled every 30s
  maxStalledCount: 2,                    // move to failed after 2 stalls
  drainDelay: 5,                         // delay between drain checks
  autorun: true,                         // start processing immediately
});
```

## Tips
- Always set `maxRetriesPerRequest: null` on IORedis connections to avoid job stalling
- Use `removeOnComplete` and `removeOnFail` to prevent Redis from growing unbounded
- Prefer BullMQ over Bull for new projects -- better API, flows, and active development
- Use sandboxed processors for CPU-intensive work to avoid blocking the event loop
- Set appropriate `lockDuration` based on your longest expected job time to prevent double processing
- Use flows (parent-child) instead of manually chaining jobs for complex dependency graphs
- Group-based rate limiting is the right pattern for multi-tenant SaaS per-customer throttling
- Always listen for `worker.on('error')` -- unhandled errors can silently drop jobs
- Use `QueueScheduler` (Bull) or ensure delayed jobs work by keeping at least one worker running
- Set Redis `maxmemory-policy` to `noeviction` -- queue data must never be silently evicted
- Use `job.updateProgress()` for long-running jobs so dashboards reflect actual progress
- Implement graceful shutdown with `worker.close()` to let active jobs finish before exit

## See Also
- redis, celery, sidekiq, rabbitmq, kafka

## References
- [BullMQ Documentation](https://docs.bullmq.io/)
- [BullMQ GitHub](https://github.com/taskforcesh/bullmq)
- [Bull GitHub (Legacy)](https://github.com/OptimalBits/bull)
- [Bull Board Dashboard](https://github.com/felixmosh/bull-board)
- [BullMQ Best Practices](https://docs.bullmq.io/guide/going-to-production)

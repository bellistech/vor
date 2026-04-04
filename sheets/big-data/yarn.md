# Apache YARN (Yet Another Resource Negotiator)

YARN is Hadoop's cluster resource management layer that decouples scheduling and resource management from data processing, enabling multiple computation frameworks like MapReduce, Spark, Flink, and Tez to share a single cluster.

## Architecture

### Core components

```text
# ResourceManager (RM)  - global master, allocates cluster resources
#   ├── Scheduler        - pure scheduling (Capacity/Fair/FIFO), no monitoring
#   └── ApplicationsManager - accepts job submissions, manages ApplicationMasters
# NodeManager (NM)      - per-node agent, launches/monitors containers
# ApplicationMaster (AM) - per-application, negotiates resources from RM
# Container             - allocation of CPU + memory on a node
# Timeline Server       - stores application history and metrics
```

## CLI Commands

### Application management

```bash
# List running applications
yarn application -list

# List applications by state
yarn application -list -appStates RUNNING,ACCEPTED,SUBMITTED
yarn application -list -appStates FINISHED -appTypes SPARK

# Get application status
yarn application -status application_1234567890_0001

# Kill an application
yarn application -kill application_1234567890_0001

# Get application logs (after completion)
yarn logs -applicationId application_1234567890_0001

# Get logs for a specific container
yarn logs -applicationId application_1234567890_0001 \
  -containerId container_1234567890_0001_01_000001

# Get logs for a specific node
yarn logs -applicationId application_1234567890_0001 \
  -nodeAddress node1:8042

# Move application to a different queue
yarn application -movetoqueue application_1234567890_0001 -queue production
```

### Cluster information

```bash
# Cluster overview
yarn cluster -lnl

# Node status
yarn node -list
yarn node -list -states RUNNING
yarn node -status node1:8042

# Cluster metrics
yarn top

# Queue information
yarn queue -status default
yarn queue -status production

# Check resource usage
yarn cluster -lnl | grep -i memory
```

### Container management

```bash
# List containers for an application
yarn container -list application_1234567890_0001

# Container status
yarn container -status container_1234567890_0001_01_000001

# Signal a container
yarn container -signal container_1234567890_0001_01_000001 \
  OUTPUT_THREAD_DUMP
```

## Queue Configuration

### Capacity Scheduler (capacity-scheduler.xml)

```xml
<!-- Root queue configuration -->
<property>
  <name>yarn.scheduler.capacity.root.queues</name>
  <value>production,development,testing</value>
</property>

<!-- Production queue: 60% capacity, can grow to 80% -->
<property>
  <name>yarn.scheduler.capacity.root.production.capacity</name>
  <value>60</value>
</property>
<property>
  <name>yarn.scheduler.capacity.root.production.maximum-capacity</name>
  <value>80</value>
</property>
<property>
  <name>yarn.scheduler.capacity.root.production.user-limit-factor</name>
  <value>2</value>
</property>

<!-- Development queue: 30% capacity -->
<property>
  <name>yarn.scheduler.capacity.root.development.capacity</name>
  <value>30</value>
</property>
<property>
  <name>yarn.scheduler.capacity.root.development.maximum-capacity</name>
  <value>50</value>
</property>

<!-- Testing queue: 10% capacity -->
<property>
  <name>yarn.scheduler.capacity.root.testing.capacity</name>
  <value>10</value>
</property>
<property>
  <name>yarn.scheduler.capacity.root.testing.maximum-capacity</name>
  <value>30</value>
</property>

<!-- Queue ACLs -->
<property>
  <name>yarn.scheduler.capacity.root.production.acl_submit_applications</name>
  <value>prod_team</value>
</property>
<property>
  <name>yarn.scheduler.capacity.root.production.acl_administer_queue</name>
  <value>admin</value>
</property>

<!-- Preemption -->
<property>
  <name>yarn.resourcemanager.scheduler.monitor.enable</name>
  <value>true</value>
</property>
<property>
  <name>yarn.resourcemanager.scheduler.monitor.policies</name>
  <value>org.apache.hadoop.yarn.server.resourcemanager.monitor.capacity.ProportionalCapacityPreemptionPolicy</value>
</property>
```

### Fair Scheduler (fair-scheduler.xml)

```xml
<?xml version="1.0"?>
<allocations>
  <defaultQueueSchedulingPolicy>fair</defaultQueueSchedulingPolicy>
  <defaultMinSharePreemptionTimeout>300</defaultMinSharePreemptionTimeout>

  <queue name="production">
    <weight>6.0</weight>
    <minResources>40960 mb, 20 vcores</minResources>
    <maxResources>81920 mb, 40 vcores</maxResources>
    <schedulingPolicy>fair</schedulingPolicy>
    <aclSubmitApps>prod_team</aclSubmitApps>
  </queue>

  <queue name="development">
    <weight>3.0</weight>
    <minResources>20480 mb, 10 vcores</minResources>
    <maxResources>61440 mb, 30 vcores</maxResources>
    <maxRunningApps>10</maxRunningApps>
    <schedulingPolicy>fair</schedulingPolicy>
  </queue>

  <queue name="testing">
    <weight>1.0</weight>
    <minResources>8192 mb, 4 vcores</minResources>
    <maxResources>40960 mb, 20 vcores</maxResources>
    <schedulingPolicy>fifo</schedulingPolicy>
  </queue>

  <!-- Queue placement rules -->
  <queuePlacementPolicy>
    <rule name="specified" />
    <rule name="user" create="false" />
    <rule name="default" queue="development" />
  </queuePlacementPolicy>
</allocations>
```

## YARN Configuration

### yarn-site.xml essentials

```xml
<!-- ResourceManager address -->
<property>
  <name>yarn.resourcemanager.address</name>
  <value>rm-host:8032</value>
</property>

<!-- ResourceManager HA -->
<property>
  <name>yarn.resourcemanager.ha.enabled</name>
  <value>true</value>
</property>
<property>
  <name>yarn.resourcemanager.ha.rm-ids</name>
  <value>rm1,rm2</value>
</property>

<!-- NodeManager resources -->
<property>
  <name>yarn.nodemanager.resource.memory-mb</name>
  <value>65536</value>
</property>
<property>
  <name>yarn.nodemanager.resource.cpu-vcores</name>
  <value>16</value>
</property>

<!-- Container memory limits -->
<property>
  <name>yarn.scheduler.minimum-allocation-mb</name>
  <value>1024</value>
</property>
<property>
  <name>yarn.scheduler.maximum-allocation-mb</name>
  <value>32768</value>
</property>
<property>
  <name>yarn.scheduler.minimum-allocation-vcores</name>
  <value>1</value>
</property>
<property>
  <name>yarn.scheduler.maximum-allocation-vcores</name>
  <value>8</value>
</property>

<!-- Container vmem check (often disabled for Spark) -->
<property>
  <name>yarn.nodemanager.vmem-check-enabled</name>
  <value>false</value>
</property>
<property>
  <name>yarn.nodemanager.pmem-check-enabled</name>
  <value>true</value>
</property>

<!-- Log aggregation -->
<property>
  <name>yarn.log-aggregation-enable</name>
  <value>true</value>
</property>
<property>
  <name>yarn.log-aggregation.retain-seconds</name>
  <value>604800</value>
</property>
<property>
  <name>yarn.nodemanager.remote-app-log-dir</name>
  <value>hdfs:///var/log/yarn/apps</value>
</property>
```

## REST API

### ResourceManager API

```bash
# Cluster info
curl http://rm-host:8088/ws/v1/cluster/info

# Cluster metrics
curl http://rm-host:8088/ws/v1/cluster/metrics

# List applications
curl http://rm-host:8088/ws/v1/cluster/apps

# Application details
curl http://rm-host:8088/ws/v1/cluster/apps/application_1234567890_0001

# List nodes
curl http://rm-host:8088/ws/v1/cluster/nodes

# Scheduler info (queue details)
curl http://rm-host:8088/ws/v1/cluster/scheduler

# Submit application via REST
curl -X POST http://rm-host:8088/ws/v1/cluster/apps/new-application
```

## Node Labels

### Label-based scheduling

```bash
# Add node labels
yarn rmadmin -addToClusterNodeLabels "gpu,ssd"

# Assign labels to nodes
yarn rmadmin -replaceLabelsOnNode "node1=gpu node2=ssd node3=gpu,ssd"

# List node labels
yarn cluster -lnl

# Configure queue to use labels (capacity-scheduler.xml)
# yarn.scheduler.capacity.root.ml-jobs.accessible-node-labels=gpu
# yarn.scheduler.capacity.root.ml-jobs.accessible-node-labels.gpu.capacity=100
```

## Tips

- Set `yarn.nodemanager.resource.memory-mb` to 80-85% of physical RAM, leaving room for OS and DataNode
- Disable `yarn.nodemanager.vmem-check-enabled` for Spark/Flink jobs; virtual memory limits cause false container kills
- Use Capacity Scheduler for multi-tenant production; Fair Scheduler for research/dev clusters with dynamic workloads
- Enable preemption to reclaim resources from over-allocated queues, but set a reasonable timeout (300s) to avoid thrashing
- Set `user-limit-factor` > 1 to let users exceed their queue's guaranteed capacity when the cluster is idle
- Always enable log aggregation; debugging failed containers without centralized logs is nearly impossible
- Monitor the ResourceManager UI at port 8088 for queue utilization, pending apps, and unhealthy nodes
- Use node labels to isolate GPU or SSD nodes for specific workloads without wasting them on general compute
- Configure `maximum-capacity` for queues to prevent a single queue from consuming the entire cluster
- Run ResourceManager in HA mode with ZooKeeper for production to avoid single point of failure
- Tune `yarn.scheduler.minimum-allocation-mb` to match your smallest workload; overly large minimums waste memory

## See Also

- hadoop, spark, hive, flink, mapreduce, zookeeper

## References

- [Apache YARN Documentation](https://hadoop.apache.org/docs/stable/hadoop-yarn/hadoop-yarn-site/YARN.html)
- [Capacity Scheduler Guide](https://hadoop.apache.org/docs/stable/hadoop-yarn/hadoop-yarn-site/CapacityScheduler.html)
- [Fair Scheduler Guide](https://hadoop.apache.org/docs/stable/hadoop-yarn/hadoop-yarn-site/FairScheduler.html)
- [YARN REST API](https://hadoop.apache.org/docs/stable/hadoop-yarn/hadoop-yarn-site/ResourceManagerRest.html)
- [YARN Commands Reference](https://hadoop.apache.org/docs/stable/hadoop-yarn/hadoop-yarn-site/YarnCommands.html)

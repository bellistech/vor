# Apache Hadoop (Distributed Storage and Processing Framework)

Hadoop provides a distributed filesystem (HDFS) and processing framework (MapReduce/YARN) for reliable, scalable big data storage and batch computation across commodity hardware clusters.

## HDFS Architecture

### Core components

```text
# NameNode      - master, manages filesystem namespace and block metadata
# DataNode      - worker, stores actual data blocks on local disk
# Secondary NN  - merges edit log with fsimage (NOT a failover NN)
# JournalNode   - shared edit log for HA NameNode setup
# BlockManager  - tracks block locations, replication, placement policy
# Rack Awareness - data placement optimized by network topology

# Default block size: 128 MB (configurable via dfs.blocksize)
# Default replication factor: 3 (dfs.replication)
```

## HDFS CLI Commands

### File operations

```bash
# List files in HDFS
hdfs dfs -ls /user/data/

# List recursively with human-readable sizes
hdfs dfs -ls -R -h /user/data/

# Create a directory
hdfs dfs -mkdir -p /user/data/input

# Upload local file to HDFS
hdfs dfs -put localfile.csv /user/data/input/

# Upload with overwrite
hdfs dfs -put -f localfile.csv /user/data/input/

# Copy from local (same as -put)
hdfs dfs -copyFromLocal data.parquet /warehouse/raw/

# Download from HDFS to local
hdfs dfs -get /user/data/output/part-00000 ./result.csv

# Copy to local (same as -get)
hdfs dfs -copyToLocal /user/data/output/ ./local_output/

# View file contents
hdfs dfs -cat /user/data/input/sample.csv | head -20

# View file tail
hdfs dfs -tail /user/data/logs/app.log

# Move/rename files
hdfs dfs -mv /user/data/old_name.csv /user/data/new_name.csv

# Remove file
hdfs dfs -rm /user/data/input/temp.csv

# Remove directory recursively
hdfs dfs -rm -r /user/data/staging/

# Skip trash on delete
hdfs dfs -rm -r -skipTrash /tmp/expired_data/

# Check disk usage
hdfs dfs -du -s -h /user/data/

# Count files and directories
hdfs dfs -count /user/data/

# Change replication factor
hdfs dfs -setrep -w 2 /user/data/archive/

# Set permissions
hdfs dfs -chmod 755 /user/data/shared/
hdfs dfs -chown hadoop:hdfs /user/data/shared/

# Merge small files into one
hdfs dfs -getmerge /user/data/output/ ./merged_output.csv

# Check file checksum
hdfs dfs -checksum /user/data/input/file.csv

# Touch (create empty file or update timestamp)
hdfs dfs -touchz /user/data/flags/_SUCCESS
```

### Snapshots

```bash
# Enable snapshots on a directory
hdfs dfsadmin -allowSnapshot /user/data/

# Create a snapshot
hdfs dfs -createSnapshot /user/data/ snap_2024_01

# List snapshots
hdfs dfs -ls /user/data/.snapshot/

# Restore from snapshot
hdfs dfs -cp /user/data/.snapshot/snap_2024_01/file.csv /user/data/file.csv

# Delete a snapshot
hdfs dfs -deleteSnapshot /user/data/ snap_2024_01
```

## Administration

### NameNode operations

```bash
# Check NameNode status
hdfs dfsadmin -report

# Safe mode operations
hdfs dfsadmin -safemode get
hdfs dfsadmin -safemode enter
hdfs dfsadmin -safemode leave

# Run filesystem check
hdfs fsck / -files -blocks -locations

# Check for corrupt or missing blocks
hdfs fsck / -list-corruptfileblocks

# Rebalance data across DataNodes
hdfs balancer -threshold 10

# Refresh node list (after adding/decommissioning nodes)
hdfs dfsadmin -refreshNodes

# Roll edit log (force checkpoint)
hdfs dfsadmin -rollEdits

# Get NameNode metrics
curl http://namenode:9870/jmx?qry=Hadoop:service=NameNode,name=FSNamesystem
```

### High Availability

```bash
# Check HA state
hdfs haadmin -getServiceState nn1
hdfs haadmin -getServiceState nn2

# Manual failover
hdfs haadmin -failover nn1 nn2

# Check health
hdfs haadmin -checkHealth nn1

# Transition to active (forced)
hdfs haadmin -transitionToActive --forceactive nn1
```

## Configuration

### hdfs-site.xml essentials

```xml
<!-- Block size (128 MB default) -->
<property>
  <name>dfs.blocksize</name>
  <value>134217728</value>
</property>

<!-- Replication factor -->
<property>
  <name>dfs.replication</name>
  <value>3</value>
</property>

<!-- NameNode data directory -->
<property>
  <name>dfs.namenode.name.dir</name>
  <value>file:///data/hdfs/namenode</value>
</property>

<!-- DataNode data directories (comma-separated for multiple disks) -->
<property>
  <name>dfs.datanode.data.dir</name>
  <value>file:///data1/hdfs/datanode,file:///data2/hdfs/datanode</value>
</property>

<!-- Rack awareness script -->
<property>
  <name>net.topology.script.file.name</name>
  <value>/etc/hadoop/rack-topology.sh</value>
</property>

<!-- NameNode handler threads -->
<property>
  <name>dfs.namenode.handler.count</name>
  <value>100</value>
</property>

<!-- Short-circuit local reads -->
<property>
  <name>dfs.client.read.shortcircuit</name>
  <value>true</value>
</property>

<!-- Trash checkpoint interval (minutes) -->
<property>
  <name>fs.trash.checkpoint.interval</name>
  <value>60</value>
</property>
```

### core-site.xml

```xml
<!-- Default filesystem URI -->
<property>
  <name>fs.defaultFS</name>
  <value>hdfs://mycluster:8020</value>
</property>

<!-- Trash retention (minutes) -->
<property>
  <name>fs.trash.interval</name>
  <value>1440</value>
</property>

<!-- IO buffer size -->
<property>
  <name>io.file.buffer.size</name>
  <value>131072</value>
</property>
```

## Rack Awareness

### Topology script example

```bash
#!/bin/bash
# /etc/hadoop/rack-topology.sh
# Maps IP/hostname to rack ID
HADOOP_CONF=/etc/hadoop/conf

while [ $# -gt 0 ]; do
  nodeArg=$1
  result=""
  if [ -f "$HADOOP_CONF/topology.data" ]; then
    result=$(grep -w "$nodeArg" "$HADOOP_CONF/topology.data" | awk '{print $2}')
  fi
  if [ -z "$result" ]; then
    echo -n "/default-rack "
  else
    echo -n "$result "
  fi
  shift
done
```

### topology.data format

```text
# hostname/IP    rack
datanode1        /rack1
datanode2        /rack1
datanode3        /rack2
datanode4        /rack2
10.0.1.10        /rack1
10.0.1.11        /rack2
```

## MapReduce

### Submitting jobs

```bash
# Run a MapReduce job
hadoop jar /opt/hadoop/share/hadoop/mapreduce/hadoop-mapreduce-examples-*.jar \
  wordcount /input/text /output/wordcount

# Run with custom config
hadoop jar myjob.jar com.example.MyJob \
  -Dmapreduce.job.reduces=10 \
  -Dmapreduce.map.memory.mb=2048 \
  /input /output

# Stream MapReduce (Python mapper/reducer)
hadoop jar /opt/hadoop/share/hadoop/tools/lib/hadoop-streaming-*.jar \
  -input /data/logs/ \
  -output /data/results/ \
  -mapper mapper.py \
  -reducer reducer.py \
  -file mapper.py \
  -file reducer.py

# Check job status
mapred job -list
mapred job -status job_1234567890_0001

# Kill a running job
mapred job -kill job_1234567890_0001

# View job history
mapred job -history /output/wordcount
```

## Benchmarks

### Built-in benchmarks

```bash
# TestDFSIO write benchmark
hadoop jar /opt/hadoop/share/hadoop/mapreduce/hadoop-mapreduce-client-jobclient-*-tests.jar \
  TestDFSIO -write -nrFiles 10 -fileSize 1GB

# TestDFSIO read benchmark
hadoop jar /opt/hadoop/share/hadoop/mapreduce/hadoop-mapreduce-client-jobclient-*-tests.jar \
  TestDFSIO -read -nrFiles 10 -fileSize 1GB

# NNBench (NameNode benchmark)
hadoop jar /opt/hadoop/share/hadoop/mapreduce/hadoop-mapreduce-client-jobclient-*-tests.jar \
  nnbench -operation create_write -maps 10 -numberOfFiles 1000

# TeraGen + TeraSort (standard cluster benchmark)
hadoop jar /opt/hadoop/share/hadoop/mapreduce/hadoop-mapreduce-examples-*.jar \
  teragen 10000000 /benchmarks/terasort-input

hadoop jar /opt/hadoop/share/hadoop/mapreduce/hadoop-mapreduce-examples-*.jar \
  terasort /benchmarks/terasort-input /benchmarks/terasort-output

hadoop jar /opt/hadoop/share/hadoop/mapreduce/hadoop-mapreduce-examples-*.jar \
  teravalidate /benchmarks/terasort-output /benchmarks/terasort-validate
```

## Tips

- Set `dfs.blocksize` to 256 MB or larger for very large files to reduce NameNode memory pressure
- Use `-getmerge` to combine MapReduce output parts into a single local file for downstream processing
- Enable short-circuit local reads (`dfs.client.read.shortcircuit=true`) for significant read performance gains on co-located tasks
- Always use `-skipTrash` when deleting large temporary datasets to avoid filling the trash directory
- Monitor NameNode heap usage closely; each block reference consumes roughly 150 bytes of heap
- Use HDFS snapshots before running destructive ETL jobs as a safety net
- Configure multiple `dfs.datanode.data.dir` entries to spread I/O across physical disks
- Run `hdfs balancer` after adding new DataNodes; set threshold to 5-10% for even distribution
- Enable rack awareness from day one; retrofitting placement policy on existing data requires rebalancing
- Use `hdfs fsck` regularly to catch under-replicated or corrupt blocks before they become data loss
- Set `dfs.namenode.handler.count` to 20x the number of DataNodes as a starting point for busy clusters
- Prefer HA NameNode with JournalNodes over Secondary NameNode for production deployments

## See Also

- yarn, spark, hive, mapreduce, hdfs, zookeeper

## References

- [Apache Hadoop Documentation](https://hadoop.apache.org/docs/stable/)
- [HDFS Architecture Guide](https://hadoop.apache.org/docs/stable/hadoop-project-dist/hadoop-hdfs/HdfsDesign.html)
- [HDFS Commands Reference](https://hadoop.apache.org/docs/stable/hadoop-project-dist/hadoop-common/FileSystemShell.html)
- [Hadoop Cluster Setup](https://hadoop.apache.org/docs/stable/hadoop-project-dist/hadoop-common/ClusterSetup.html)
- [HDFS High Availability](https://hadoop.apache.org/docs/stable/hadoop-project-dist/hadoop-hdfs/HDFSHighAvailabilityWithQJM.html)

# SAN Storage -- Storage Area Network Architecture and Protocols

> A Storage Area Network (SAN) is a dedicated, high-speed network that interconnects and presents shared pools of storage devices to multiple servers. Unlike Network Attached Storage (NAS), which operates at the file level over standard Ethernet, a SAN provides raw block-level access, allowing each host to treat remote storage as locally attached disks. This document covers the foundational theory, protocol mechanics, and design principles that underpin modern SAN environments.

---

## 1. Storage Access Models: DAS, NAS, and SAN

Understanding where SANs fit requires comparing the three primary storage access models.

**Direct Attached Storage (DAS)** is the simplest model. A disk or array connects directly to a single server via SAS, SATA, or NVMe. The server owns the storage exclusively. DAS offers the lowest latency because there is no network hop, but it creates storage silos. If server A has 2 TB free and server B is full, there is no mechanism to share that capacity without recabling or migrating data.

**Network Attached Storage (NAS)** solves the sharing problem by placing a dedicated appliance on the Ethernet network. Clients access files through protocols like NFS (Unix/Linux) or SMB/CIFS (Windows). The NAS appliance owns the filesystem and manages permissions, locking, and caching. NAS is ideal for unstructured data — home directories, media files, shared documents. However, NAS operates at the file level, which introduces overhead unsuitable for latency-sensitive workloads like databases.

**Storage Area Networks (SAN)** provide block-level access over a dedicated network. Each server sees SAN-presented storage as a local SCSI or NVMe device and creates its own filesystem on it. This gives applications the performance characteristics of local disk with the flexibility of networked storage. SANs excel at structured, transactional workloads: relational databases, virtual machine hypervisors, email servers, and any application that issues small random I/O at high rates.

The fundamental distinction is the access granularity. Block storage (DAS, SAN) lets the host manage the filesystem. File storage (NAS) delegates filesystem management to the appliance. Object storage (S3, Swift) abandons hierarchical filesystems entirely in favor of flat namespaces with rich metadata — a model suited for cloud-native applications and massive-scale unstructured data.

---

## 2. SCSI Fundamentals

The Small Computer System Interface (SCSI) architecture is the foundation upon which Fibre Channel and iSCSI SANs are built. Even NVMe borrows conceptual elements from SCSI while redesigning the command set for parallelism.

### 2.1 The SCSI Architectural Model

SCSI defines a client-server relationship. The **initiator** sends commands and the **target** processes them and returns responses. A target exposes one or more **Logical Unit Numbers (LUNs)**, each representing an addressable storage volume. When a host issues a read or write, it constructs a **Command Descriptor Block (CDB)** containing the operation code, logical block address, and transfer length. The target executes the command against the specified LUN and returns status.

This model is transport-independent. The same CDB format works whether the SCSI commands travel over a parallel SCSI bus, a Fibre Channel link, an iSCSI TCP connection, or even USB (via UAS). The transport layer encapsulates the CDB and routes it to the correct target.

### 2.2 LUN Addressing and Masking

A LUN is not a physical disk; it is a logical abstraction. A storage array may have hundreds of physical disks organized into RAID groups or pools, carved into LUNs of varying sizes. **LUN masking** is a security mechanism that restricts which initiators can see which LUNs. The array's access control list (ACL) maps initiator identifiers (WWPNs for FC, IQNs for iSCSI) to specific LUNs. Without masking, any initiator on the fabric could access any LUN, which would be catastrophic in a multi-tenant environment.

### 2.3 SCSI Reservations

When multiple hosts access a shared LUN (as in a cluster), they need a coordination mechanism to prevent data corruption. SCSI-2 introduced simple reservations, but these were single-host and fragile. **SCSI-3 Persistent Reservations (PR)** replaced them with a robust system supporting multiple registrants. Each host registers a key with the LUN, and reservation types define access rules (Write Exclusive, Exclusive Access, Write Exclusive - Registrants Only, etc.). Cluster software like Windows Failover Clustering and Linux pacemaker rely on SCSI-3 PR for fencing and I/O fencing.

---

## 3. Fibre Channel SAN

Fibre Channel (FC) was purpose-built for storage networking. It provides lossless, ordered, in-order delivery of frames — properties that TCP provides but at significant overhead. FC achieves low latency by implementing flow control and error detection in hardware.

### 3.1 FC Protocol Stack

FC defines five layers. **FC-0** is the physical layer: laser optics, multimode or single-mode fiber, and copper twinax for short distances. **FC-1** handles encoding. Older generations (1G through 8G) used 8b/10b encoding, which adds 25% overhead. 16G and above switched to 64b/66b encoding, reducing overhead to ~3%. **FC-2** is the framing and signaling layer, responsible for frame construction, buffer-to-buffer credits, and flow control. **FC-3** defines common services like multicast and hunt groups, though these are rarely used in practice. **FC-4** is the upper-layer protocol mapping — most commonly FCP (Fibre Channel Protocol for SCSI), which maps SCSI commands into FC frames.

### 3.2 FC Addressing

Every FC port has a **World Wide Port Name (WWPN)**, a globally unique 64-bit identifier assigned at manufacture (analogous to a MAC address). Each node also has a **World Wide Node Name (WWNN)**. When a port logs into a fabric, the switch assigns it a 24-bit **FC address** (domain:area:port format). The FC Name Server maintains the mapping between WWPNs and FC addresses, allowing initiators to discover targets by name.

### 3.3 Fabric Services

An FC switch provides several built-in services. The **Fabric Login Server (FLOGI)** authenticates devices connecting to the fabric. The **Name Server** maintains a directory of all logged-in devices and their attributes. The **Registered State Change Notification (RSCN)** service alerts devices when the fabric topology changes (a new device logs in, a port goes down). The **Zone Server** enforces access control by segmenting the fabric into zones.

### 3.4 Zoning Theory

Zoning is the FC equivalent of firewall rules. It controls which initiators can communicate with which targets. Without zoning, every device on the fabric can see every other device, which creates security risks and RSCN storms (every topology change notifies every device).

**Soft zoning** is enforced by the Name Server. When an initiator queries the Name Server, it only receives information about devices in its zone. However, if an initiator knows the FC address of a device outside its zone, it can still communicate with it. **Hard zoning** is enforced in the switch hardware (ASIC). Frames destined for a port outside the zone are dropped at the hardware level, regardless of how the initiator obtained the address. Production environments should always use hard zoning.

The recommended practice is **single-initiator zoning**: each zone contains exactly one host HBA and one or more target ports. This prevents host-to-host communication on the storage fabric and limits the blast radius of RSCN notifications. When a target port goes offline, only the hosts in zones containing that port receive the notification.

### 3.5 Inter-Switch Links and Fabric Design

Switches connect to each other via **Inter-Switch Links (ISLs)**. Trunking combines multiple physical ISLs into a single logical link for bandwidth aggregation and redundancy. FC fabrics support two topologies: **core-edge** (smaller environments with edge switches connecting to a core) and **spine-leaf** (larger environments with full mesh between spine and leaf tiers, minimizing hop count).

The **domain ID** uniquely identifies each switch in the fabric. The maximum domain count depends on the switch vendor and firmware — typically 239 domains per fabric. Fabric merges occur when two isolated fabrics are connected; if domain IDs conflict, the merge fails and one fabric is segmented. This is why documentation and planning are essential before connecting switches.

---

## 4. iSCSI

iSCSI (Internet Small Computer System Interface) encapsulates SCSI commands inside TCP/IP packets, allowing block storage access over standard Ethernet networks. This dramatically lowers the cost of entry compared to FC, since no specialized HBAs or switches are required.

### 4.1 Protocol Mechanics

An iSCSI session begins with **discovery**. The initiator sends a **SendTargets** request to a known portal (IP:port, default 3260). The target responds with a list of available target names and their portal addresses. The initiator then performs a **login** to establish a session with a specific target. During login, parameters are negotiated: authentication method, header/data digest preferences, maximum burst lengths, and connection count.

Once the session is established, SCSI commands are serialized into **iSCSI PDUs (Protocol Data Units)** and sent over one or more TCP connections. Each PDU has an iSCSI header containing the command sequence number, transfer tag, and other metadata. The target processes the command and returns a response PDU with status and data.

### 4.2 Authentication with CHAP

The Challenge-Handshake Authentication Protocol (CHAP) provides authentication for iSCSI sessions. **One-way CHAP** means the target challenges the initiator to prove its identity. The target sends a challenge (random value), the initiator hashes it with the shared secret and returns the response, and the target verifies it. **Mutual CHAP** adds reverse authentication: after the target authenticates the initiator, the initiator challenges the target. This prevents man-in-the-middle attacks where a rogue device impersonates a legitimate target.

CHAP is not encryption. It only authenticates identity. For data confidentiality in transit, IPsec can be layered underneath iSCSI, though this adds CPU overhead and is rarely deployed in practice. Most organizations rely on network isolation (dedicated VLANs, physical separation) rather than encryption for iSCSI security.

### 4.3 iSCSI Network Design

iSCSI is sensitive to network quality. Unlike FC, which has hardware-level flow control and lossless delivery, Ethernet is best-effort. TCP retransmissions and congestion cause latency spikes and throughput degradation. To mitigate this:

**Dedicated storage VLANs** isolate iSCSI traffic from general network traffic. Storage I/O should never compete with user browsing or email.

**Jumbo frames (MTU 9000)** reduce CPU overhead by allowing larger payloads per frame. Every device in the path — host NIC, every switch, and the target — must support the same MTU. A single device with MTU 1500 in the path silently fragments frames, destroying performance.

**iSCSI HBAs (TOE cards)** offload TCP/IP processing to dedicated hardware, reducing host CPU usage. Software initiators work well for moderate workloads, but high-throughput environments benefit from hardware offload.

**Multiple connections per session (MC/S)** and **multiple sessions** provide bandwidth aggregation and failover. Combined with multipathing, this delivers redundancy comparable to dual-fabric FC.

---

## 5. NVMe over Fabrics (NVMe-oF)

NVMe (Non-Volatile Memory Express) was designed from the ground up for flash storage, replacing the SCSI command set with a leaner, parallelism-oriented architecture. NVMe over Fabrics extends NVMe beyond the local PCIe bus to remote storage over a network.

### 5.1 Why NVMe-oF Exists

SCSI was designed in an era of spinning disks with millisecond latencies. Its command processing model is serial: one outstanding command per queue, with the operating system managing a single hardware queue. NVMe introduces **multiple submission and completion queues** (up to 65,535 queues, each with 65,536 entries), allowing massive parallelism. The command set is simplified to a handful of opcodes optimized for flash. NVMe-oF preserves these advantages over a network fabric, targeting microsecond-level latencies rather than the milliseconds typical of iSCSI or FC.

### 5.2 Transport Options

**NVMe/FC** runs NVMe commands over existing Fibre Channel infrastructure. Organizations with FC investments can adopt NVMe-oF without replacing switches. The FC switch must support NVMe/FC (most modern firmware does), and both host HBAs and array ports must support the NVMe/FC initiator and target roles.

**NVMe/RDMA** uses Remote Direct Memory Access (via RoCE v2 or InfiniBand) to transfer data directly between host and target memory without CPU involvement. This delivers the lowest latency of any NVMe-oF transport but requires RDMA-capable NICs and lossless Ethernet configuration (PFC, ECN).

**NVMe/TCP** encapsulates NVMe commands in TCP, similar to how iSCSI encapsulates SCSI. It requires no special hardware — any TCP/IP NIC works. While latency is higher than RDMA, NVMe/TCP is far simpler to deploy and is rapidly becoming the default choice for environments that do not already have FC or RDMA infrastructure.

### 5.3 Asymmetric Namespace Access (ANA)

ANA is the NVMe equivalent of SCSI ALUA. In a multi-controller array, a namespace may be accessible through multiple controllers, but the path through the owning controller is "optimized" (lowest latency), while paths through non-owning controllers are "non-optimized" (the request is internally forwarded). The host multipath driver reads ANA state and prefers optimized paths, failing over to non-optimized paths if the preferred controller goes down.

---

## 6. Multipathing

Multipathing is the practice of providing multiple physical paths between a host and a storage LUN. If one path fails (HBA failure, cable break, switch failure, controller failover), I/O transparently shifts to a surviving path. Beyond redundancy, multipathing provides load balancing across paths.

### 6.1 Path States and Policies

Each path to a LUN has a state:

- **Active/Optimized**: The preferred path through the owning controller. I/O flows directly to the LUN with minimum latency.
- **Active/Non-Optimized**: A path through a non-owning controller. I/O works but is internally proxied to the owning controller, adding latency.
- **Standby**: The path is valid but not currently carrying I/O. Used for failover only.
- **Unavailable**: The path is down (link failure, controller offline).

Path selection policies determine how I/O is distributed:

- **Round-Robin**: I/O is distributed equally across all active paths. Suitable for Active/Active arrays.
- **Least Queue Depth**: I/O goes to the path with the fewest outstanding commands. Good for heterogeneous path performance.
- **Service Time**: I/O goes to the path with the lowest estimated service time. Adapts to congestion.

### 6.2 ALUA (Asymmetric Logical Unit Access)

ALUA is the SCSI standard (SPC-3) that allows an array to report path priorities to the host. The array advertises Target Port Group (TPG) states — optimized, non-optimized, standby, unavailable — for each LUN. The host multipath driver queries these states and routes I/O accordingly. When a controller failover occurs, the array updates TPG states, the host detects the change, and the multipath driver adjusts routing. This is the standard mechanism used by virtually all modern arrays (Dell, HPE, NetApp, Pure, Hitachi).

### 6.3 Linux Device Mapper Multipath

The Linux `dm-multipath` subsystem creates a single `/dev/mapper/mpathX` device from multiple `/dev/sdX` paths. The `multipathd` daemon monitors path health, handles failover, and rebalances I/O. Configuration in `/etc/multipath.conf` defines path grouping policies, failback behavior, and vendor-specific tuning. The `find_multipaths` option prevents multipath from claiming non-SAN devices (local disks, USB drives), which is critical for system stability.

---

## 7. Thin Provisioning

Thin provisioning decouples the capacity presented to hosts from the physical capacity consumed on disk. A storage pool of 10 TB can present 50 TB of virtual capacity to hosts. Space is allocated from the pool only when data is actually written.

### 7.1 How It Works

The array maintains a pool of physical extents (small chunks, typically 4 KB to 16 MB). When a host writes to a thin LUN for the first time, the array allocates extents from the pool to back that write. Regions of the LUN that have never been written consume zero physical capacity. This eliminates the traditional problem of over-allocating storage "just in case" and dramatically improves utilization rates.

### 7.2 Space Reclamation

When a host deletes files, the filesystem marks blocks as free in its metadata, but the array has no visibility into filesystem-level operations. The array still considers those blocks allocated. **SCSI UNMAP** (the block protocol equivalent of ATA TRIM) allows the host to inform the array that specific LBAs are no longer in use. The array then returns those extents to the pool. Without UNMAP, thin pools gradually fill to 100% even though hosts have deleted significant data.

### 7.3 Risks and Monitoring

The critical risk of thin provisioning is pool exhaustion. If the physical pool reaches 100% utilization, all LUNs in that pool stop accepting writes. This is a shared-fate failure — an out-of-space condition on a single LUN affects every other LUN in the pool. Monitoring tools must track pool utilization and trigger alerts well before exhaustion (70%, 80%, 90% thresholds are standard). Over-provisioning ratios should be planned based on actual write patterns, not arbitrary targets.

---

## 8. Snapshots

A snapshot is a point-in-time image of a LUN. It captures the state of all blocks at the moment of creation, allowing recovery of data as it existed at that instant.

### 8.1 Copy-on-Write (CoW)

When a snapshot is taken, no data is copied immediately. The snapshot simply records the current block map. When a subsequent write occurs to the source LUN, the original block is first copied to a snapshot reserve area, then the new data overwrites the source block. Reads of the snapshot are satisfied from the snapshot reserve (for changed blocks) or the source LUN (for unchanged blocks). CoW snapshots are fast to create but impose a write penalty (each write requires a read-copy-write sequence).

### 8.2 Redirect-on-Write (RoW)

Instead of copying the original block before overwriting it, RoW redirects new writes to a different location. The snapshot retains pointers to the original blocks, which remain in place. This avoids the read-modify-write penalty of CoW but can cause data fragmentation over time as blocks scatter across the storage pool.

### 8.3 Consistency Groups

Databases and applications that span multiple LUNs require all LUNs to be snapped at the same instant. A **consistency group** coordinates snapshot creation across multiple LUNs, ensuring a crash-consistent or application-consistent image. Without consistency groups, LUN A might be snapped one second before LUN B, creating an inconsistent dataset that cannot be recovered.

---

## 9. Replication

Replication copies data from a primary site to a secondary site for disaster recovery. The two fundamental modes are synchronous and asynchronous.

### 9.1 Synchronous Replication

Every write to the primary LUN is simultaneously sent to the remote replica. The write is not acknowledged to the host until both copies are committed. This guarantees **RPO = 0** (Recovery Point Objective of zero data loss). The cost is latency: every write incurs the round-trip delay to the remote site. At the speed of light in fiber (~5 microseconds per kilometer), a 100 km distance adds 1 millisecond of latency per write. Beyond ~200 km, the latency penalty is generally unacceptable for transaction-heavy workloads.

### 9.2 Asynchronous Replication

Writes are committed locally and replicated to the remote site in the background. The host sees local write latency only, with no distance penalty. The trade-off is **RPO > 0**: data written between replication cycles is lost if the primary site fails. The RPO depends on the replication interval and available WAN bandwidth. If the remote site is 30 seconds behind the primary and a disaster strikes, up to 30 seconds of data is lost.

### 9.3 Three-Site Topologies

A common enterprise design uses three sites: Site A and Site B are in the same metro area with synchronous replication (zero data loss). Site B asynchronously replicates to Site C in a different region (disaster recovery for regional events). This provides both zero RPO for local failures and geographic protection for catastrophic events.

---

## 10. Storage Tiering

Storage tiering is the practice of placing data on different classes of storage media based on performance requirements and access frequency. In a SAN environment, tiering happens within the array and is transparent to the host.

### 10.1 Tier Definitions

**Tier 0** consists of NVMe or SSD media. These devices deliver the highest IOPS (hundreds of thousands to millions per device) and the lowest latency (sub-millisecond). They are used for the most demanding workloads: OLTP databases, real-time analytics, and latency-sensitive applications. Cost per gigabyte is highest at this tier.

**Tier 1** uses high-speed spinning disks, typically 15K or 10K RPM SAS drives. While vastly outperformed by flash, these drives still serve workloads that need reasonable random I/O performance but cannot justify the cost of all-flash. This tier is increasingly rare as flash prices decline.

**Tier 2** employs high-capacity, low-RPM drives — 7.2K RPM NL-SAS or SATA. These drives optimize for sequential throughput and cost per terabyte. They are suited for backups, archives, bulk data storage, and workloads that are predominantly sequential.

**Tier 3** extends to tape libraries, object storage, and cloud archive services. Data at this tier may take seconds to minutes to retrieve and is accessed rarely — regulatory archives, historical backups, and cold data.

### 10.2 Automated Tiering

Modern storage arrays implement sub-LUN auto-tiering, which operates at a granularity of small extents (typically 256 KB to 1 MB chunks). The array continuously monitors I/O patterns and builds a "heat map" of block-level activity. Hot blocks (frequently accessed) are promoted to faster tiers; cold blocks (rarely accessed) are demoted to cheaper tiers. This promotion and demotion happens automatically and transparently, without host awareness.

The key advantage of sub-LUN tiering is that a single LUN can span multiple tiers simultaneously. A database LUN might have its actively queried indexes on Tier 0 flash, its frequently accessed tables on Tier 1 SAS, and its historical partitions on Tier 2 NL-SAS. The administrator provisions one LUN and the array optimizes placement continuously.

### 10.3 Policy-Based Tiering

Some environments require deterministic placement rather than algorithmic auto-tiering. Policy-based tiering allows administrators to pin specific LUNs or applications to designated tiers. For example, a compliance requirement might mandate that database transaction logs always reside on Tier 0 flash for write performance, regardless of how frequently they are read.

---

## 11. RAID in the SAN Context

RAID (Redundant Array of Independent Disks) protects against disk failures within the storage array. While RAID is not unique to SANs, understanding RAID behavior is essential for SAN administrators because the RAID level directly affects LUN performance characteristics, rebuild risk, and usable capacity.

**RAID 1 (Mirroring)** duplicates every block to a second disk. It provides excellent read performance (reads can be served from either disk) and fast rebuilds (simply copy one disk to another). The cost is 50% capacity loss. RAID 1 is used for small, performance-critical LUNs such as database logs.

**RAID 5 (Single Parity)** distributes data and a single parity block across all disks in the group. Capacity efficiency is (N-1)/N — a 5-disk RAID 5 group yields 80% usable space. However, RAID 5 has a critical weakness with modern large-capacity drives. Rebuilding a failed 16 TB drive requires reading every block from every surviving disk, which takes many hours. During that rebuild window, a second disk failure is catastrophic — all data in the group is lost. The probability of a second failure during rebuild increases with drive size and rebuild duration. For these reasons, RAID 5 is considered obsolete for enterprise SAN use and should be avoided for any production workload.

**RAID 6 (Double Parity)** adds a second independent parity calculation, allowing the group to survive two simultaneous disk failures. Capacity efficiency is (N-2)/N. Write performance is slightly worse than RAID 5 (two parity calculations per write). RAID 6 is the minimum acceptable RAID level for spinning disk arrays in production SAN environments.

**RAID 10 (Striped Mirrors)** combines mirroring and striping. Data is first mirrored (RAID 1) and then striped (RAID 0) across mirror pairs. Each mirror pair can lose one disk without data loss. RAID 10 provides the best write performance (no parity calculation) and the fastest rebuild times (only the mirror partner needs copying). The trade-off is 50% capacity overhead. RAID 10 is preferred for write-intensive workloads such as OLTP databases.

**Distributed and Wide-Stripe RAID** is the modern approach used by all-flash arrays. Traditional RAID groups are small (4-8 disks), creating hot spots and limiting rebuild parallelism. Distributed RAID spreads data and parity across all drives in a storage pool, sometimes dozens or hundreds of drives. When a drive fails, every surviving drive participates in the rebuild, completing it in minutes rather than hours. Examples include NetApp RAID-DP and RAID-TEC, HPE ADAPT, Dell distributed RAID, and Pure Storage VRAID.

---

## 12. SAN Design Principles

### 12.1 Dual-Fabric Architecture

The cardinal rule of SAN design is complete redundancy with fault isolation. A production SAN uses two independent fabrics (Fabric A and Fabric B) with no physical or logical interconnection between them. Each host has two HBAs, one connected to each fabric. Each array controller presents ports to both fabrics. A complete failure of Fabric A (switch failure, firmware bug, human error) leaves Fabric B fully operational. The host multipath driver transparently routes all I/O to the surviving fabric.

### 12.2 Zoning Strategy

Every production fabric should enforce single-initiator zoning with hard zoning enabled. Zone names should follow a consistent convention that encodes the host name, HBA port, array name, and array port. Device aliases map every WWPN to a human-readable name. The zoning configuration should be version-controlled or at minimum documented in a spreadsheet that is updated with every change.

### 12.3 Performance Planning

SAN performance depends on multiple factors: HBA queue depth, switch buffer credits, ISL oversubscription ratio, array controller cache hit rate, and backend disk performance. The **oversubscription ratio** is the ratio of edge port bandwidth to ISL bandwidth. A 2:1 ratio means the edge ports can theoretically generate twice the traffic that the ISLs can carry. For latency-sensitive workloads, a 1:1 ratio (no oversubscription) is preferred.

**Buffer credits** control FC flow control. Each credit represents permission to send one frame. The number of credits needed between two ports depends on the distance and link speed: `credits = (distance_km * link_speed_Gbps) / (frame_size * speed_of_light_in_fiber)`. Insufficient credits cause **credit starvation**, where a port stops sending because it has no credits left, even though the link has available bandwidth. Long-distance ISLs and extended fabrics are particularly susceptible.

### 12.4 iSCSI-Specific Design

iSCSI networks should be isolated on dedicated VLANs or, preferably, dedicated physical switches. Jumbo frames (MTU 9000) must be configured end-to-end — host NIC, every switch in the path, and the storage target. A single device at MTU 1500 in the middle will cause fragmentation and massive performance degradation that is difficult to diagnose.

Quality of Service (QoS) settings should prioritize iSCSI traffic. If iSCSI shares a network with other traffic (not recommended but sometimes unavoidable in converged infrastructure), DSCP markings and priority queuing prevent storage I/O from being starved during network congestion.

### 12.5 NVMe-oF Considerations

NVMe/TCP deployments benefit from the same network design principles as iSCSI: dedicated networks, jumbo frames, and QoS. NVMe/RDMA requires lossless Ethernet, which means configuring Priority Flow Control (PFC) on specific traffic classes and Explicit Congestion Notification (ECN) for congestion management. RDMA is intolerant of packet drops; a single dropped frame triggers expensive recovery mechanisms that negate the latency advantage.

For NVMe/FC, existing FC fabric design practices apply. The primary consideration is firmware compatibility: both host HBAs and array ports must support the NVMe/FC protocol, and the FC switches must run firmware versions that can route NVMe frames.

---

## Prerequisites

- Understanding of TCP/IP networking fundamentals (subnets, VLANs, routing)
- Familiarity with Linux block device concepts (`/dev/sdX`, filesystems, mount points)
- Basic knowledge of disk I/O concepts (IOPS, throughput, latency)
- Understanding of RAID concepts at a theoretical level
- For Fibre Channel sections: awareness of optical networking basics (SFPs, fiber types)
- For NVMe-oF/RDMA: familiarity with RDMA concepts (RoCE, InfiniBand, lossless Ethernet)

---

## References

- INCITS T10 Technical Committee — SCSI Standards: https://www.t10.org/
- INCITS T11 Technical Committee — Fibre Channel Standards: https://www.t11.org/
- NVM Express Specifications: https://nvmexpress.org/specifications/
- SNIA (Storage Networking Industry Association) — Tutorials and dictionary: https://www.snia.org/
- RFC 7143 — Internet Small Computer System Interface (iSCSI) Protocol (Consolidated): https://www.rfc-editor.org/rfc/rfc7143
- RFC 3720 — Internet Small Computer System Interface (iSCSI): https://www.rfc-editor.org/rfc/rfc3720
- RFC 7145 — Internet Small Computer System Interface Extensions for RDMA (iSER): https://www.rfc-editor.org/rfc/rfc7145
- SCSI Primary Commands (SPC-5) — ALUA and Persistent Reservations: https://www.t10.org/
- NVMe over Fabrics Specification: https://nvmexpress.org/specifications/
- Linux Kernel NVMe Target Documentation: https://docs.kernel.org/nvme/index.html
- Red Hat Enterprise Linux — Configuring DM Multipath: https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/
- Brocade Fabric OS Administration Guide: https://www.broadcom.com/
- Cisco MDS 9000 Series Configuration Guides: https://www.cisco.com/
- "Storage Area Networks For Dummies" (Wiley) — Christopher Poelker, Alex Nikitin
- "SAN and NAS Storage Networking" — W. Curtis Preston

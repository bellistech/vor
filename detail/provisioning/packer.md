# The Mathematics of Packer — Machine Image Building Theory

> *Packer automates the creation of machine images for multiple platforms from a single configuration. Its execution model involves builder parallelism, provisioner sequencing, post-processor pipelines, and the economics of image-based vs. boot-time provisioning.*

---

## 1. Build Parallelism (Multi-Platform Execution)

### The Problem

Packer can build images for multiple platforms simultaneously. The execution model determines total build time.

### Parallel Build Formula

$$T_{total} = \max_{b \in \text{builds}} T_b$$

Where each build's time:

$$T_b = T_{launch} + T_{provision} + T_{snapshot} + T_{cleanup}$$

### Worked Example

| Build | Launch | Provision | Snapshot | Total |
|:---|:---:|:---:|:---:|:---:|
| AWS AMI | 60s | 300s | 120s | 480s = 8 min |
| GCP Image | 45s | 300s | 90s | 435s = 7.25 min |
| Azure VHD | 90s | 300s | 180s | 570s = 9.5 min |
| Docker | 5s | 300s | 10s | 315s = 5.25 min |

$$T_{parallel} = \max(480, 435, 570, 315) = 570\text{s} = 9.5 \text{ min}$$

$$T_{sequential} = 480 + 435 + 570 + 315 = 1{,}800\text{s} = 30 \text{ min}$$

$$\text{Speedup} = \frac{30}{9.5} = 3.16\times$$

### Maximum Theoretical Speedup

$$\text{Speedup}_{max} = \frac{\sum T_b}{\max T_b}$$

---

## 2. Provisioner Pipeline (Sequential Execution)

### The Problem

Within a single build, provisioners run sequentially. The order matters — each builds on the previous.

### Provisioner Chain

$$S_{final} = p_N(p_{N-1}(\cdots p_2(p_1(S_{base}))\cdots))$$

Where $S_{base}$ = base image state, $p_i$ = provisioner $i$.

### Total Provisioning Time

$$T_{provision} = \sum_{i=1}^{N} T_{p_i}$$

### Worked Example: Web Server Image

| Provisioner | Type | Time | Cumulative |
|:---|:---|:---:|:---:|
| Update OS | shell | 60s | 60s |
| Install packages | shell | 45s | 105s |
| Copy configs | file | 5s | 110s |
| Configure nginx | shell | 10s | 120s |
| Run tests | shell | 30s | 150s |
| Cleanup | shell | 15s | 165s |

$$T_{provision} = 165\text{s} = 2.75 \text{ min}$$

### Provisioner Failure Impact

If provisioner $k$ fails:

$$\text{Wasted time} = T_{launch} + \sum_{i=1}^{k} T_{p_i}$$

For a failure at step 5 of the above: $60 + 120 = 180\text{s}$ wasted (including launch).

---

## 3. Image vs. Boot-Time Provisioning (Cost-Time Tradeoff)

### The Problem

Should you bake software into an image (Packer) or provision at boot time (cloud-init/Ansible)? This is an optimization problem.

### The Cost Model

**Image-based (Packer):**

$$C_{image} = C_{build} + N \times T_{boot} \times C_{compute}$$

**Boot-time provisioning:**

$$C_{boot} = N \times (T_{boot} + T_{provision}) \times C_{compute}$$

### Break-Even Analysis

Image-based is cheaper when:

$$C_{build} < N \times T_{provision} \times C_{compute}$$

$$N > \frac{C_{build}}{T_{provision} \times C_{compute}}$$

### Worked Example

- Build cost: $C_{build} = \$0.50$ (10 min on c5.large at $0.05/min)
- Provision time: $T_{provision} = 300\text{s} = 5\text{ min}$
- Instance cost: $C_{compute} = \$0.01/\text{min}$ (t3.small)

$$N > \frac{0.50}{5 \times 0.01} = 10 \text{ instances}$$

At 10+ instances, pre-baking the image saves money.

### Time-to-Serve Comparison

| Method | Launch | Provision | Serve | Total |
|:---|:---:|:---:|:---:|:---:|
| Boot-time | 30s | 300s | 330s | 5.5 min |
| Pre-baked image | 30s | 0s | 30s | 0.5 min |

**11x faster to serve traffic** with a pre-baked image.

---

## 4. Post-Processor Pipelines

### The Problem

Post-processors transform or distribute built artifacts. They can be chained or run in parallel.

### Pipeline Model

$$\text{artifact}_{final} = pp_k(\cdots pp_2(pp_1(\text{artifact}_{build}))\cdots)$$

### Common Pipelines

| Pipeline | Steps | Output |
|:---|:---|:---|
| AMI → compress → upload | 3 | S3 archive |
| Docker → tag → push | 3 | Registry image |
| VirtualBox → vagrant → upload | 3 | Vagrant Cloud box |
| AMI → manifest | 2 | JSON manifest |

### Post-Processor Timing

$$T_{post} = \sum_{pp \in \text{pipeline}} T_{pp}$$

| Post-Processor | Typical Time |
|:---|:---:|
| manifest | < 1s |
| docker-tag | < 1s |
| docker-push | 30-300s (depends on layers) |
| compress | 30-120s (depends on size) |
| vagrant-cloud upload | 60-600s |

---

## 5. Image Size Optimization

### The Problem

Smaller images boot faster, cost less to store, and transfer more quickly.

### Size Factors

$$S_{image} = S_{OS} + S_{packages} + S_{data} + S_{cache} - S_{cleanup}$$

### Cleanup Savings

| Cleanup Step | Savings |
|:---|:---:|
| `apt-get clean` | 100-500 MB |
| Remove build deps | 200-1000 MB |
| Clear logs | 10-100 MB |
| Zero free space (+ compress) | 30-70% of free space |
| Remove man pages/docs | 50-200 MB |

### Zeroing and Compression

$$S_{compressed} = S_{used} + S_{zero\_blocks} \times R_{compression}$$

Where $R_{compression} \approx 0.001$ for zero blocks (they compress almost perfectly).

**Before zeroing:** 10 GB image with 3 GB used, 7 GB random free space → 10 GB compressed ≈ 8 GB.

**After zeroing:** 10 GB image with 3 GB used, 7 GB zeroed → compressed ≈ 3.1 GB.

$$\text{Savings} = 1 - \frac{3.1}{8} = 61.3\%$$

---

## 6. Template Variables and Functions

### The Problem

HCL2 templates support variables, functions, and data sources. Understanding evaluation order prevents build failures.

### Variable Precedence

$$V_{final} = V_{default} \triangleleft V_{var\_file} \triangleleft V_{env} \triangleleft V_{cli}$$

### Timestamp-Based Naming

$$\text{image\_name} = \text{prefix-}\text{formatdate}(\text{"YYYYMMDDHHmmss"}, \text{timestamp()})$$

### Image Naming Uniqueness

For builds running every hour:

$$P(\text{collision}) = 0 \quad \text{(timestamp resolution is 1 second)}$$

For concurrent builds:

$$\text{name} = \text{prefix-}\text{uuidv4()} \implies P(\text{collision}) \approx \frac{n^2}{2^{122}}$$

---

## 7. Multi-Cloud Image Matrix

### The Problem

Organizations need images across multiple regions, OS versions, and cloud providers.

### Image Matrix Size

$$|I| = |\text{OS versions}| \times |\text{providers}| \times |\text{regions}|$$

| Dimension | Count |
|:---|:---:|
| OS versions | 3 (Ubuntu 22.04, 24.04, Amazon Linux 2023) |
| Cloud providers | 3 (AWS, GCP, Azure) |
| Regions per provider | 5 |

$$|I| = 3 \times 3 \times 5 = 45 \text{ images}$$

### Build Time (Fully Parallel)

$$T_{matrix} = \max_{i \in I} T_i$$

### Monthly Cost

$$C_{monthly} = |I| \times S_{image} \times R_{storage} + N_{builds} \times C_{build}$$

For 45 images at 10 GB, $0.05/GB/month, rebuilt weekly:

$$C = 45 \times 10 \times 0.05 + 4 \times 45 \times 0.50 = 22.50 + 90 = \$112.50/\text{month}$$

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\max T_b$ | Maximum | Parallel build time |
| $\sum T_{p_i}$ | Summation | Provisioner pipeline |
| $N > C_{build}/(T_{prov} \times C_{comp})$ | Break-even | Image vs boot-time |
| $S_{used} + S_{zero} \times R_{comp}$ | Compression | Image size |
| $|OS| \times |Cloud| \times |Region|$ | Cartesian product | Image matrix |
| $V_{def} \triangleleft V_{file} \triangleleft V_{env}$ | Override algebra | Variable precedence |

---

*Packer turns "works on my machine" into "works on every cloud" — the multi-platform parallel build model, image optimization math, and break-even analysis make the case for pre-baked images over boot-time provisioning at any scale beyond a handful of instances.*

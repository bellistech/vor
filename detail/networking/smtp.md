# The Mathematics of SMTP — Queuing Theory & Delivery Probability

> *Every mail server is a queue processor whose delivery success depends on retry backoff schedules, reputation scoring functions, and the probabilistic alignment of authentication signals — SPF, DKIM, and DMARC form a Boolean satisfiability problem for every inbound message.*

---

## 1. Mail Queue as an M/G/1 System (Queueing Theory)

### The Problem

A mail server receives messages at a stochastic rate and processes (delivers) them with variable service times depending on remote server responsiveness, DNS lookups, and TLS negotiation. What is the expected queue length and waiting time?

### The Formula

Model the mail server as an M/G/1 queue (Poisson arrivals, general service distribution, single delivery thread per destination):

$$L_q = \frac{\rho^2 + \lambda^2 \sigma_s^2}{2(1 - \rho)}$$

Where:
- $\lambda$ = arrival rate (messages/second)
- $\mu$ = service rate (deliveries/second), $E[S] = 1/\mu$
- $\rho = \lambda / \mu$ = server utilization
- $\sigma_s^2$ = variance of service time
- $L_q$ = mean number of messages waiting in queue

The mean waiting time (Pollaczek-Khinchine formula):

$$W_q = \frac{\rho \cdot E[S] + \lambda \sigma_s^2}{2(1 - \rho)}$$

Total time in system:

$$W = W_q + E[S]$$

### Worked Examples

**Example 1:** A small mail server receives 2 messages/sec, delivers at average 0.4 sec/message ($\mu = 2.5$/sec), with service time standard deviation 0.3 sec.

$$\rho = \frac{2}{2.5} = 0.8$$

$$W_q = \frac{0.8 \times 0.4 + 2 \times 0.09}{2(1 - 0.8)} = \frac{0.32 + 0.18}{0.4} = 1.25 \text{ sec}$$

$$W = 1.25 + 0.4 = 1.65 \text{ sec}$$

**Example 2:** At $\rho = 0.95$ (near saturation), same variance:

$$W_q = \frac{0.95 \times 0.4 + 2 \times 0.09}{2(0.05)} = \frac{0.38 + 0.18}{0.1} = 5.6 \text{ sec}$$

At 95% utilization, queue wait jumps to 5.6 sec per message — illustrating the nonlinear growth as $\rho \to 1$.

---

## 2. Retry Backoff and Delivery Probability (Geometric Processes)

### The Problem

When delivery fails with a 4xx temporary error, the MTA retries with exponential backoff. What is the probability of successful delivery within $n$ retries if each attempt has independent success probability $p$?

### The Formula

Probability of delivery by attempt $k$:

$$P(\text{delivered by attempt } k) = 1 - (1 - p)^k$$

Expected number of attempts until delivery:

$$E[N] = \frac{1}{p}$$

With exponential backoff, the $k$-th retry occurs at time:

$$t_k = t_0 \cdot b^{k-1}$$

Where $t_0$ is the initial retry interval and $b$ is the backoff multiplier. Total elapsed time after $n$ retries:

$$T_n = t_0 \cdot \frac{b^n - 1}{b - 1}$$

### Worked Examples

**Example:** Postfix defaults: $t_0 = 300$ sec (5 min), $b = 2$, max 5 days. If $p = 0.7$ per attempt:

$$P(\text{delivered by 3 attempts}) = 1 - (1 - 0.7)^3 = 1 - 0.027 = 0.973$$

Time elapsed after 3 retries:

$$T_3 = 300 \cdot \frac{2^3 - 1}{2 - 1} = 300 \cdot 7 = 2100 \text{ sec} = 35 \text{ min}$$

After 8 retries:

$$T_8 = 300 \cdot \frac{256 - 1}{1} = 76500 \text{ sec} \approx 21.25 \text{ hours}$$

$$P(\text{delivered by 8 attempts}) = 1 - 0.3^8 = 1 - 0.000066 \approx 0.99993$$

---

## 3. SPF/DKIM/DMARC as Boolean Satisfiability (Authentication Logic)

### The Problem

DMARC alignment requires that at least one of SPF or DKIM passes AND aligns with the From domain. This creates a Boolean formula that determines message disposition.

### The Formula

Let:
- $S$ = SPF passes, $S_a$ = SPF aligns (envelope From domain matches header From)
- $D$ = DKIM passes, $D_a$ = DKIM aligns (d= domain matches header From)

DMARC pass:

$$\text{DMARC} = (S \land S_a) \lor (D \land D_a)$$

If SPF and DKIM are independent with probabilities $p_s, p_{sa}, p_d, p_{da}$:

$$P(\text{DMARC pass}) = 1 - (1 - p_s \cdot p_{sa})(1 - p_d \cdot p_{da})$$

### Worked Examples

**Example:** A well-configured domain has $p_s = 0.95$, $p_{sa} = 0.98$, $p_d = 0.99$, $p_{da} = 1.0$ (DKIM always aligns when it passes):

$$P(\text{SPF aligned}) = 0.95 \times 0.98 = 0.931$$

$$P(\text{DKIM aligned}) = 0.99 \times 1.0 = 0.99$$

$$P(\text{DMARC pass}) = 1 - (1 - 0.931)(1 - 0.99) = 1 - 0.069 \times 0.01 = 1 - 0.00069 = 0.99931$$

With both mechanisms, DMARC failure is extremely rare (< 0.07%). If DKIM is disabled ($p_d = 0$):

$$P(\text{DMARC pass}) = p_s \cdot p_{sa} = 0.931$$

Losing 7% of legitimate mail to DMARC failure — a strong argument for always deploying both.

---

## 4. IP Reputation Scoring (Logistic Regression Model)

### The Problem

Major email providers use reputation scoring to classify sending IPs. The score is typically modeled as a logistic function of behavioral signals.

### The Formula

$$R(x) = \frac{1}{1 + e^{-(\beta_0 + \sum_{i=1}^{n} \beta_i x_i)}}$$

Where signals $x_i$ include:
- $x_1$ = bounce rate (negative weight)
- $x_2$ = complaint rate (negative weight)
- $x_3$ = authentication pass rate (positive weight)
- $x_4$ = sending volume consistency (positive weight)
- $x_5$ = age of IP (positive weight)

### Worked Examples

**Example:** Simplified model with $\beta_0 = 0$, three signals:

| Signal | $x_i$ | $\beta_i$ | Contribution |
|--------|--------|-----------|-------------|
| Bounce rate | 0.02 | -50 | -1.0 |
| Auth pass rate | 0.99 | 5 | 4.95 |
| Complaint rate | 0.001 | -200 | -0.2 |

$$z = -1.0 + 4.95 - 0.2 = 3.75$$

$$R = \frac{1}{1 + e^{-3.75}} = \frac{1}{1 + 0.0235} = 0.977$$

High reputation (97.7%). If bounce rate rises to 8%:

$$z = -4.0 + 4.95 - 0.2 = 0.75$$

$$R = \frac{1}{1 + e^{-0.75}} = \frac{1}{1 + 0.472} = 0.679$$

Reputation drops to 67.9% — likely triggering spam folder placement.

---

## 5. Message Size and Transfer Time (Shannon Capacity)

### The Problem

SMTP transfers are bounded by TCP throughput. For a message of size $M$ bytes over a link with bandwidth $B$ and round-trip time RTT, what is the minimum transfer time?

### The Formula

With TCP window scaling, effective throughput:

$$\text{Throughput} = \min\left(B, \frac{W_{\max}}{\text{RTT}}\right)$$

SMTP adds overhead from command/response round trips. Each transaction requires at minimum 5 round trips (connect, EHLO, MAIL FROM, RCPT TO, DATA):

$$T_{smtp} = 5 \cdot \text{RTT} + \frac{M}{\text{Throughput}} + T_{tls}$$

Where $T_{tls} \approx 2 \cdot \text{RTT}$ for TLS 1.3 handshake (1-RTT mode).

### Worked Examples

**Example:** 10 MB message, 100 Mbps link, 50 ms RTT, 64KB TCP window:

$$\text{Throughput} = \min\left(100 \text{ Mbps}, \frac{65536 \times 8}{0.05}\right) = \min(100, 10.49) = 10.49 \text{ Mbps}$$

$$T_{smtp} = 5 \times 0.05 + \frac{10 \times 8}{10.49} + 2 \times 0.05 = 0.25 + 7.63 + 0.10 = 7.98 \text{ sec}$$

With window scaling to 1 MB:

$$\text{Throughput} = \min(100, 167.8) = 100 \text{ Mbps}$$

$$T_{smtp} = 0.25 + 0.8 + 0.10 = 1.15 \text{ sec}$$

Window scaling reduces transfer time by 7x for large messages.

---

## 6. Greylisting Effectiveness (Bayesian Filtering)

### The Problem

Greylisting rejects unknown senders with a 4xx code, expecting legitimate MTAs to retry but spam bots to give up. What is the false positive / false negative rate?

### The Formula

Using Bayes' theorem, the probability a retrying sender is legitimate:

$$P(\text{legit} | \text{retry}) = \frac{P(\text{retry} | \text{legit}) \cdot P(\text{legit})}{P(\text{retry})}$$

Where:

$$P(\text{retry}) = P(\text{retry} | \text{legit}) \cdot P(\text{legit}) + P(\text{retry} | \text{spam}) \cdot P(\text{spam})$$

### Worked Examples

**Example:** 10% of incoming is legitimate, 90% spam. Legitimate MTAs retry 99% of the time, spam bots retry 15%:

$$P(\text{retry}) = 0.99 \times 0.1 + 0.15 \times 0.9 = 0.099 + 0.135 = 0.234$$

$$P(\text{legit} | \text{retry}) = \frac{0.099}{0.234} = 0.423$$

Greylisting reduced spam from 90% to 57.7% of accepted mail — a 36% improvement in the spam-to-legitimate ratio without any content analysis.

---

## Prerequisites

- Queueing theory (M/G/1, Pollaczek-Khinchine formula)
- Probability (geometric distribution, Bayes' theorem)
- Boolean logic (satisfiability, logical operators)
- Logistic regression basics
- TCP/IP fundamentals (windowing, RTT, throughput)
- DNS (MX records, TXT records)

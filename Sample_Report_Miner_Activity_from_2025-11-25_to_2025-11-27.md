# Cryptocurrency Mining Impact Assessment Report

**Report Generated:** 2025-12-01 21:32:07 PST

**Analysis Period:** 2025-11-25 08:00:00 to 2025-11-27 08:00:00 (2d 0h 0m)

---

## Executive Summary

This report documents the impact of unauthorized cryptocurrency mining operations on company resources.
The analysis covers **5 unique systems** across **5 unique IP addresses** that were actively participating in
mining operations.

### Key Findings

- **Total Mining Time:** 212.00 hours (8.83 days)
- **Total Computational Work:** 19.68B hashes computed
- **Estimated Energy Consumption:** 36.87 kWh
- **Estimated Electricity Cost:** $4.42
- **Blocks Mined:** 187
- **Systems with High CPU Usage (>80%):** 2 of 5 (40.0%)
- **GPU Mining Enabled:** 1 systems
- **Hybrid CPU+GPU Mining:** 1 systems

---

## 1. Resource Consumption Analysis

### Compute Power Impact

The unauthorized mining operations consumed significant computational resources:

- **Average System CPU Utilization:** 76.8%
- **Peak CPU Utilization Observed:** 95.8%
- **Total CPU-Hours Consumed:** 212.00 hours
- **Mining Type Distribution:**
  - CPU Only: 4 systems
  - GPU Accelerated: 0 systems
  - Hybrid CPU+GPU: 1 systems

**Impact:** Mining operations typically maximize processor utilization, severely degrading performance for
legitimate workloads. Systems experiencing >80% CPU usage would have experienced significant slowdowns,
delayed processing times, and poor user experience.

### Electricity Consumption

- **Total Energy Consumed:** 36.87 kWh
- **Estimated Cost:** $4.42 (at $0.120 per kWh)
- **Daily Average:** 4.17 kWh/day ($0.50/day)

**Methodology:** Energy estimates based on average CPU power consumption of 150W and GPU power consumption
of 250W under sustained load, adjusted for actual CPU utilization percentages.

**Impact:** Continuous high-utilization operations result in substantial electricity costs. This represents
pure waste, as the computational work provided no business value.

### Network Impact

- **Total Blocks Mined:** 187
- **Mining Pool Connections:** Persistent connections to external mining pool infrastructure
- **Data Transfer:** Ongoing work requests and solution submissions

**Impact:** While mining is not extremely bandwidth-intensive, it generates persistent outbound connections
that may have bypassed security monitoring and created potential data exfiltration channels.

---

## 2. Performance Impact Assessment

### System Degradation

Based on CPU utilization levels, affected systems experienced the following performance impacts:

| Impact Level | Systems | CPU Usage Range | Expected User Experience |
|--------------|---------|-----------------|---------------------------|
| **CRITICAL** | 2 | 90-100% | System severely degraded, frequent hangs, near-unusable |
| **HIGH** | 1 | 70-89% | Significant slowdowns, delayed response times |
| **MODERATE** | 2 | 50-69% | Noticeable performance reduction, slower operations |
| **LOW** | 0 | 0-49% | Minor impact, occasional slowdowns |

### Service Availability

**Productivity Loss:** Users on affected systems would have experienced:

- Increased application load times
- Slower document processing and file operations
- Delayed response to user input
- Potential application timeouts or crashes due to resource starvation
- Reduced multitasking capability

**Business Impact:** For systems categorized as HIGH or CRITICAL impact (3 systems), productivity loss
likely reached 40-70%, representing significant business disruption.

---

## 3. Infrastructure Damage Assessment

### Hardware Wear and Lifespan Impact

Sustained high-utilization mining operations accelerate hardware degradation:

**Component Stress:**

- **Processors (CPU/GPU):** Running at maximum load for 212.00 total hours
- **Cooling Systems:** Fans running at maximum speed to dissipate heat
- **Power Supplies:** Operating under continuous high load
- **Motherboards:** Sustained high current draw through VRM components

**Expected Lifespan Reduction:**

- Normal enterprise hardware lifespan: 5-7 years
- Estimated lifespan reduction from sustained mining: 20-40%
- Accelerated replacement costs for 5 affected systems

### Thermal Stress

**Risk Factors:**

- Sustained high temperatures degrade silicon and solder joints
- Increased risk of thermal shutdowns and component failure
- Potential for thermal paste degradation requiring maintenance
- Systems with GPU mining (1 systems) face additional thermal stress

**Recommendation:** All affected systems should undergo thermal inspection and preventive maintenance,
including thermal paste replacement and cooling system verification.

---

## 4. Security Implications

### Breach Indicators

The presence of cryptocurrency mining software indicates a security compromise:

**Affected Infrastructure:**

- **Total Compromised Systems:** 5
- **Unique Hostnames:** 5
- **Unique IP Addresses:** 5
- **Mining Period:** 2d 0h 0m

**Deployment Vectors (Requires Investigation):**

- Malware infection (trojan, worm, or targeted attack)
- Compromised credentials (employee accounts or service accounts)
- Insider threat (authorized user deploying unauthorized software)
- Supply chain compromise (infected software or updates)
- Vulnerable services or unpatched systems

### Persistence Mechanisms

**Operational Characteristics:**

- Mining clients maintained persistent connections to external pool servers
- Total operational time: 212.00 hours suggests robust persistence mechanisms
- Systems likely configured for auto-start and restart after interruption

**Access Level:**

- Mining software requires sufficient privileges to consume resources
- Likely has user-level or higher access on all 5 affected systems
- Potential for lateral movement if network credentials were compromised

**Broader Exposure Risk:**

- If mining software was deployed via compromised credentials, attacker may still have access
- Systems should be forensically analyzed for additional malware or backdoors
- Network traffic logs should be reviewed for data exfiltration attempts
- All affected systems require credential rotation and security hardening

---

## 5. Financial Impact Summary

### Direct Costs

| Cost Category | Estimated Amount | Notes |
|---------------|------------------|-------|
| Electricity Costs | $4.42 | Based on 36.87 kWh at $0.120/kWh |
| Accelerated Hardware Replacement | $1.50K | Estimated 20-40% lifespan reduction |
| Incident Response | $6.00K | Security investigation, forensics, remediation |
| System Maintenance | $750.00 | Thermal inspection, cleaning, repairs |

### Indirect Costs

**Productivity Loss:**

- 3 systems experienced significant performance degradation
- Estimated productivity loss: 40-70% on severely affected systems
- User time wasted on slow systems and rework

**Operational Impact:**

- IT staff time spent investigating and remediating
- Potential SLA violations if critical services were affected
- Management time spent on incident response coordination

**Reputational Risk:**

- Security breach may require disclosure depending on industry regulations
- Potential loss of customer confidence
- Regulatory implications if sensitive data was accessed

---

## 6. Detailed System-by-System Impact Analysis

The following table shows impact metrics for each compromised system, sorted by estimated cost (highest impact first):

| Hostname | IP Address | Mining Type | Mining Time | Avg CPU | Hashes | Blocks | Est. kWh | Est. Cost | Impact Level |
|----------|------------|-------------|-------------|---------|--------|--------|----------|-----------|---------------|
| GAMING-PC-02 | 192.168.1.145 | CPU+GPU Hybrid | 40.0h | 65.2% | 4.3B | 42 | 16.00 | $1.92 | MODERATE |
| SERVER-DB-01 | 10.0.5.12 | CPU Only | 45.0h | 95.8% | 5.9B | 52 | 6.47 | $0.78 | CRITICAL |
| WORKSTATION-01 | 192.168.1.105 | CPU Only | 46.0h | 92.5% | 4.8B | 45 | 6.38 | $0.77 | CRITICAL |
| LAPTOP-HR-05 | 192.168.1.87 | CPU Only | 43.0h | 78.3% | 3.5B | 38 | 5.05 | $0.61 | HIGH |
| FINANCE-WS-11 | 10.0.8.23 | CPU Only | 38.0h | 52.1% | 1.1B | 10 | 2.97 | $0.36 | MODERATE |

### Top 10 Highest Impact Systems

#### 1. GAMING-PC-02

- **IP Address:** 192.168.1.145
- **Mining Type:** CPU+GPU Hybrid
- **First Seen:** 2025-11-25 12:03:20
- **Last Seen:** 2025-11-27 07:59:55
- **Total Mining Time:** 1d 16h 0m
- **Average CPU Usage:** 65.2%
- **Total Hashes Computed:** 4.32B
- **Blocks Mined:** 42
- **Estimated Energy:** 16.00 kWh
- **Estimated Cost:** $1.92
- **Impact Assessment:** MODERATE - Noticeable slowdowns

#### 2. SERVER-DB-01

- **IP Address:** 10.0.5.12
- **Mining Type:** CPU Only
- **First Seen:** 2025-11-25 09:22:15
- **Last Seen:** 2025-11-25 09:22:15
- **Total Mining Time:** 1d 21h 0m
- **Average CPU Usage:** 95.8%
- **Total Hashes Computed:** 5.92B
- **Blocks Mined:** 52
- **Estimated Energy:** 6.47 kWh
- **Estimated Cost:** $0.78
- **Impact Assessment:** CRITICAL - System severely degraded

#### 3. WORKSTATION-01

- **IP Address:** 192.168.1.105
- **Mining Type:** CPU Only
- **First Seen:** 2025-11-25 08:15:23
- **Last Seen:** 2025-11-25 14:30:00
- **Total Mining Time:** 1d 22h 0m
- **Average CPU Usage:** 92.5%
- **Total Hashes Computed:** 4.85B
- **Blocks Mined:** 45
- **Estimated Energy:** 6.38 kWh
- **Estimated Cost:** $0.77
- **Impact Assessment:** CRITICAL - System severely degraded

#### 4. LAPTOP-HR-05

- **IP Address:** 192.168.1.87
- **Mining Type:** CPU Only
- **First Seen:** 2025-11-25 10:40:00
- **Last Seen:** 2025-11-27 07:57:45
- **Total Mining Time:** 1d 19h 0m
- **Average CPU Usage:** 78.3%
- **Total Hashes Computed:** 3.54B
- **Blocks Mined:** 38
- **Estimated Energy:** 5.05 kWh
- **Estimated Cost:** $0.61
- **Impact Assessment:** HIGH - Significant performance impact

#### 5. FINANCE-WS-11

- **IP Address:** 10.0.8.23
- **Mining Type:** CPU Only
- **First Seen:** 2025-11-25 13:26:40
- **Last Seen:** 2025-11-27 07:56:20
- **Total Mining Time:** 1d 14h 0m
- **Average CPU Usage:** 52.1%
- **Total Hashes Computed:** 1.05B
- **Blocks Mined:** 10
- **Estimated Energy:** 2.97 kWh
- **Estimated Cost:** $0.36
- **Impact Assessment:** MODERATE - Noticeable slowdowns

---

## 7. Recommendations and Next Steps

### Immediate Actions (Within 24 Hours)

1. **Isolate affected systems** - Disconnect all 5 compromised systems from the network
2. **Terminate mining processes** - Stop all cryptocurrency mining operations
3. **Preserve evidence** - Take forensic images of critical systems before remediation
4. **Reset credentials** - Force password reset for all user accounts on affected systems
5. **Review network logs** - Analyze traffic for data exfiltration or lateral movement

### Short-Term Actions (Within 1 Week)

1. **Forensic analysis** - Conduct detailed investigation to determine breach vector
2. **Malware removal** - Clean or reimage all 5 affected systems
3. **Security patching** - Apply all critical security updates across infrastructure
4. **Deploy EDR/monitoring** - Install endpoint detection and response tools
5. **Network segmentation** - Review and implement proper network isolation
6. **User awareness** - Brief employees on the incident and security best practices

### Long-Term Actions (Within 1 Month)

1. **Security audit** - Comprehensive review of all security controls
2. **Hardware inspection** - Thermal analysis and preventive maintenance on affected systems
3. **Policy enforcement** - Implement application whitelisting and stricter security policies
4. **Monitoring enhancement** - Deploy outbound connection monitoring for mining pool traffic
5. **Backup verification** - Ensure backups are clean and not compromised
6. **Incident response plan** - Update IR procedures based on lessons learned

### Preventive Measures

- Implement application whitelisting to prevent unauthorized software execution
- Deploy network monitoring to detect mining pool connections
- Enable CPU usage alerting to detect abnormal resource consumption
- Enforce principle of least privilege for user accounts
- Regular security awareness training for all employees
- Maintain aggressive patch management schedule

---

## Appendix

### Methodology and Assumptions

**Data Source:** Mining pool server log file (`pool_log.json`)

**Energy Consumption Calculations:**

- CPU Power Consumption: 150W average under full load
- GPU Power Consumption: 250W average under full load
- Hybrid Mode: Combined CPU + GPU power consumption
- Adjusted by actual CPU usage percentage for CPU-only systems
- Electricity rate: $0.120 per kWh

**Impact Level Classifications:**

- CRITICAL: CPU usage ≥90%
- HIGH: CPU usage 70-89%
- MODERATE: CPU usage 50-69%
- LOW: CPU usage <50%

### Unique Systems Breakdown

- **Total Unique Miners:** 5
- **Unique Hostnames:** 5
- **Unique IP Addresses:** 5

---

## End of Report

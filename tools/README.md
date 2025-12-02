# RedTeamCoin Analysis Tools

This directory contains tools for analyzing mining pool activity and generating impact assessment reports.

## Damage Assessment Report Generator

The `generate_report` tool analyzes mining pool log files and generates comprehensive damage assessment reports focused on the impact of cryptocurrency mining on company resources.

### Building the Tool

```bash
# From the project root directory
make build-tools
```

This creates `bin/generate_report`.

### Usage

```bash
./bin/generate_report -log <path_to_pool_log.json>
```

**Example:**
```bash
./bin/generate_report -log pool_log.json
```

The tool will generate a markdown report file named:
```
Report_Miner_Activity_from_<start_date>_to_<end_date>.md
```

### Report Contents

The generated report includes comprehensive analysis across seven main sections:

#### 1. Executive Summary
- Key findings and metrics
- Total mining time and computational work
- Energy consumption estimates
- Number of affected systems

#### 2. Resource Consumption Analysis
- **Compute Power Impact**
  - CPU/GPU utilization statistics
  - Mining type distribution (CPU, GPU, Hybrid)
  - Performance degradation assessment
- **Electricity Consumption**
  - Total energy consumed (kWh)
  - Estimated costs based on electricity rates
  - Daily consumption averages
- **Network Impact**
  - Mining pool connections
  - Data transfer patterns

#### 3. Performance Impact Assessment
- **System Degradation Categories**
  - CRITICAL (90-100% CPU usage)
  - HIGH (70-89% CPU usage)
  - MODERATE (50-69% CPU usage)
  - LOW (0-49% CPU usage)
- **Service Availability**
  - Productivity loss estimates
  - User experience impact
  - Business disruption assessment

#### 4. Infrastructure Damage Assessment
- **Hardware Wear and Lifespan Impact**
  - Component stress analysis
  - Expected lifespan reduction (20-40%)
  - Replacement cost implications
- **Thermal Stress**
  - Heat-related damage risks
  - Maintenance recommendations

#### 5. Security Implications
- **Breach Indicators**
  - Affected infrastructure inventory
  - Potential deployment vectors
  - Investigation requirements
- **Persistence Mechanisms**
  - Access level assessment
  - Lateral movement risks
  - Broader exposure concerns

#### 6. Financial Impact Summary
- **Direct Costs**
  - Electricity costs
  - Hardware replacement costs
  - Incident response expenses
  - System maintenance costs
- **Indirect Costs**
  - Productivity loss
  - Operational impact
  - Reputational risk

#### 7. Detailed System-by-System Analysis
- Individual miner impact metrics
- Top 10 highest impact systems
- Detailed per-system breakdowns including:
  - Mining type and duration
  - CPU usage statistics
  - Energy consumption
  - Cost estimates
  - Impact assessment

#### 8. Recommendations and Next Steps
- **Immediate Actions** (Within 24 hours)
- **Short-Term Actions** (Within 1 week)
- **Long-Term Actions** (Within 1 month)
- **Preventive Measures**

### Cost Assumptions

The report uses the following default assumptions for cost calculations:

| Parameter | Default Value | Notes |
|-----------|---------------|-------|
| CPU Power Consumption | 150W | Average under full load |
| GPU Power Consumption | 250W | Average under full load |
| Electricity Rate | $0.12/kWh | Adjust for your region |
| Avg Workstation Cost | $1,000 | For replacement estimates |
| Hardware Lifespan Reduction | 20-40% | From sustained mining |
| Incident Response Base Cost | $5,000 | Initial investigation |
| Per-System Remediation | $200 | Forensics and cleanup |
| Thermal Maintenance | $150/system | Inspection and repairs |

To customize these values, edit the constants in `tools/generate_report.go`:
- `avgCPUPowerWatts`
- `avgGPUPowerWatts`
- `electricityCostPer`

### Report Metrics Explained

**Mining Time:** Total time each system spent actively mining, calculated from heartbeat data.

**CPU Usage:** Average CPU utilization percentage during mining operations.

**Total Hashes:** Number of cryptographic hash computations performed.

**Blocks Mined:** Number of valid blocks found and submitted to the blockchain.

**Energy Consumption:** Calculated as:
```
Energy (kWh) = (Power Watts × Mining Hours) / 1000
```

For CPU-only systems, power is adjusted by CPU usage percentage. For GPU and hybrid systems, full power consumption is assumed.

**Estimated Cost:** Energy consumption × electricity rate per kWh.

**Impact Level:** Based on average CPU usage:
- **CRITICAL:** ≥90% - System severely degraded, near-unusable
- **HIGH:** 70-89% - Significant slowdowns, major productivity impact
- **MODERATE:** 50-69% - Noticeable performance reduction
- **LOW:** <50% - Minor impact, occasional slowdowns

### Example Report

A sample report is generated when you run the tool against the included `sample_pool_log.json`:

```bash
./bin/generate_report -log sample_pool_log.json
```

This demonstrates the report format and provides a template for real incident analysis.

### Use Cases

1. **Post-Incident Analysis:** Generate comprehensive damage reports after detecting and removing cryptocurrency miners
2. **Executive Briefings:** Provide clear, non-technical summaries of security incidents for management
3. **Financial Justification:** Document costs for budget requests for security improvements
4. **Compliance Documentation:** Create audit trails for regulatory requirements
5. **Insurance Claims:** Provide detailed evidence for cybersecurity insurance claims
6. **Legal Evidence:** Generate documented reports for potential legal action

### Integration with RedTeamCoin Server

The RedTeamCoin server automatically generates `pool_log.json` with detailed mining activity. To analyze this log:

1. Stop the mining pool server
2. Locate the `pool_log.json` file in the server directory
3. Run the report generator:
   ```bash
   ./bin/generate_report -log pool_log.json
   ```
4. Review the generated markdown report

### Customization

To customize the report for your organization:

1. **Adjust cost parameters:** Edit constants in `generate_report.go`
2. **Modify impact thresholds:** Change `highCPUThreshold` and impact level calculations
3. **Add custom sections:** Extend the report structure in `writeMarkdownReport()`
4. **Custom branding:** Add company headers, logos (as markdown), or footer information

### Output Format

Reports are generated in **Markdown format** for maximum compatibility:
- Easily converted to PDF, HTML, or DOCX
- Version control friendly (diff/track changes)
- Readable in any text editor
- Compatible with documentation systems (Confluence, GitHub, GitLab)
- Easy to customize and brand

### Converting Reports

To convert the markdown report to other formats:

**PDF (using pandoc):**
```bash
pandoc Report_Miner_Activity_from_2025-11-25_to_2025-11-27.md -o report.pdf
```

**HTML:**
```bash
pandoc Report_Miner_Activity_from_2025-11-25_to_2025-11-27.md -o report.html
```

**DOCX (Microsoft Word):**
```bash
pandoc Report_Miner_Activity_from_2025-11-25_to_2025-11-27.md -o report.docx
```

## Future Tools

Additional analysis tools planned for this directory:
- Network traffic analyzer for mining pool connections
- Real-time monitoring dashboard
- Automated alerting based on CPU usage thresholds
- Historical trend analysis and visualization

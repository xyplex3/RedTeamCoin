# HTTPS/TLS Setup Guide

This guide explains how to configure and use HTTPS/TLS encryption for the RedTeamCoin server.

## Overview

RedTeamCoin supports both HTTP and HTTPS modes for the REST API:

- **HTTP Mode** (default): Runs on port 8080, no encryption
- **HTTPS Mode**: Runs on port 8443 with TLS encryption, HTTP redirect on port 8080

## Quick Start

### 1. Generate TLS Certificates

Run the certificate generation script:

```bash
./generate_certs.sh
```

This creates:

- `certs/server.crt` - Self-signed TLS certificate
- `certs/server.key` - Private key

**Note**: These are self-signed certificates for development/testing. For production, use certificates from a trusted CA.

### 2. Start Server with HTTPS

```bash
export RTC_USE_TLS=true
./bin/server
```

Or in one command:

```bash
RTC_USE_TLS=true ./bin/server
```

### 3. Access the Dashboard

The server will display the HTTPS URL in the console:

```text
https://localhost:8443?token=YOUR_TOKEN_HERE
```

**Browser Warning**: Since the certificate is self-signed, browsers will show a security warning:

1. Click "Advanced"
2. Click "Proceed to localhost" (or similar)
3. Dashboard will load

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RTC_USE_TLS` | Enable HTTPS/TLS | `false` |
| `RTC_CERT_FILE` | Path to TLS certificate | `certs/server.crt` |
| `RTC_KEY_FILE` | Path to TLS private key | `certs/server.key` |
| `RTC_AUTH_TOKEN` | API authentication token | Auto-generated |

### Using Custom Certificates

If you have your own TLS certificates:

```bash
export RTC_USE_TLS=true
export RTC_CERT_FILE="/path/to/your/certificate.crt"
export RTC_KEY_FILE="/path/to/your/private.key"
./bin/server
```

### Combined Configuration

```bash
export RTC_USE_TLS=true
export RTC_AUTH_TOKEN="my-secure-token-here"
export RTC_CERT_FILE="certs/server.crt"
export RTC_KEY_FILE="certs/server.key"
./bin/server
```

## Port Configuration

### HTTP Mode (Default)

- REST API: `http://localhost:8080`
- gRPC: `localhost:50051`

### HTTPS Mode

- REST API: `https://localhost:8443`
- HTTP Redirect: `http://localhost:8080` → redirects to HTTPS
- gRPC: `localhost:50051`

## API Access with HTTPS

### Using curl

Accept self-signed certificates with `-k` flag:

```bash
# Pool statistics
curl -k -H "Authorization: Bearer YOUR_TOKEN" https://localhost:8443/api/stats

# Miners list
curl -k -H "Authorization: Bearer YOUR_TOKEN" https://localhost:8443/api/miners

# Blockchain data
curl -k -H "Authorization: Bearer YOUR_TOKEN" https://localhost:8443/api/blockchain
```

### Using Python requests

```python
import requests

# Disable SSL verification for self-signed certs
headers = {'Authorization': 'Bearer YOUR_TOKEN'}
response = requests.get('https://localhost:8443/api/stats',
                       headers=headers,
                       verify=False)
print(response.json())
```

### Using JavaScript/Fetch

The web dashboard automatically handles the token when using the URL parameter.

For custom scripts:

```javascript
const token = 'YOUR_TOKEN_HERE';
const headers = { 'Authorization': `Bearer ${token}` };

fetch('https://localhost:8443/api/stats', { headers })
  .then(r => r.json())
  .then(data => console.log(data));
```

## Testing Scripts

### Bash Script

```bash
export RTC_USE_TLS=true
export RTC_AUTH_TOKEN="your-token-here"
./examples/api_test.sh
```

### Python Script

```bash
export RTC_USE_TLS=true
export RTC_AUTH_TOKEN="your-token-here"
./examples/api_test.py
```

## Troubleshooting

### Certificate Not Found Error

**Error**: `TLS is enabled but certificates not found!`

**Solution**: Generate certificates:

```bash
./generate_certs.sh
```

### Permission Denied on Certificate Files

**Error**: Permission denied reading certificate files

**Solution**: Check file permissions:

```bash
chmod 644 certs/server.crt
chmod 600 certs/server.key
```

### Browser Refuses Connection

**Problem**: Browser shows "This site can't provide a secure connection"

**Solutions**:

1. Verify server is running with `RTC_USE_TLS=true`
2. Check you're using the correct port (8443 for HTTPS)
3. Try using the HTTP redirect URL first (http://localhost:8080)

### curl SSL Certificate Problem

**Error**: `SSL certificate problem: self signed certificate`

**Solution**: Use `-k` or `--insecure` flag:

```bash
curl -k https://localhost:8443/api/stats
```

### Python SSL Certificate Verify Failed

**Error**: `SSLError: [SSL: CERTIFICATE_VERIFY_FAILED]`

**Solution**: Set `verify=False` in requests:

```python
response = requests.get(url, headers=headers, verify=False)
```

Or disable warnings:

```python
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
```

## Security Considerations

### Self-Signed Certificates

The included certificate generation script creates **self-signed certificates**:

**✓ Pros:**

- Free and easy to generate
- Provides encryption for development/testing
- No external dependencies

**✗ Cons:**

- Not trusted by browsers (shows warnings)
- Not suitable for production
- No identity verification

### Production Certificates

For production use, obtain certificates from a trusted CA:

**Options:**

1. **Let's Encrypt** (Free)
   - Use `certbot` to obtain free certificates
   - Automatic renewal
   - Trusted by all browsers

2. **Commercial CA** (Paid)
   - DigiCert, GlobalSign, etc.
   - Extended validation options
   - Warranty/support

3. **Internal CA** (Enterprise)
   - Use your organization's certificate authority
   - Trusted within your network

### Certificate Management

**Best Practices:**

- Keep private keys secure (permissions 600)
- Rotate certificates before expiration
- Store private keys outside version control
- Use strong key sizes (minimum 2048-bit RSA or 256-bit ECDSA)
- Monitor certificate expiration dates

### Additional Security

For enhanced security, consider:

- Implement certificate rotation
- Add mutual TLS (mTLS) for client authentication
- Use hardware security modules (HSM) for key storage
- Implement certificate pinning for known clients
- Add rate limiting and request throttling

## Certificate Details

The generated self-signed certificate includes:

```text
Subject: C=US, ST=State, L=City, O=RedTeamCoin, OU=Mining, CN=localhost
Subject Alternative Names: DNS:localhost, IP:127.0.0.1
Key Algorithm: RSA 4096-bit
Validity: 365 days
```

## Next Steps

After setting up HTTPS:

1. Test API endpoints with authentication
2. Verify HTTP to HTTPS redirect works
3. Configure firewall rules if needed
4. Set up monitoring and logging
5. Plan certificate renewal strategy

## Support

For issues or questions about TLS/HTTPS configuration:

- Check server console logs for error messages
- Verify environment variables are set correctly
- Ensure certificates are in the correct location
- Review this guide's troubleshooting section

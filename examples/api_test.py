#!/usr/bin/env python3
"""
Example Python script for testing RedTeamCoin API with authentication
"""

import os
import requests
import json
import urllib3

# Disable SSL warnings for self-signed certificates
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

# Get token from environment or use placeholder
TOKEN = os.getenv('RTC_AUTH_TOKEN', 'your-token-here')
USE_TLS = os.getenv('RTC_SERVER_TLS_ENABLED', 'false').lower() == 'true'

# Set API URL based on TLS setting
if USE_TLS:
    API_URL = 'https://localhost:8443'
    VERIFY_SSL = False  # Accept self-signed certificates
else:
    API_URL = 'http://localhost:8080'
    VERIFY_SSL = True

# Set up headers with authentication
headers = {
    'Authorization': f'Bearer {TOKEN}',
    'Content-Type': 'application/json'
}

def print_section(title):
    print(f"\n{'='*60}")
    print(f"{title}")
    print('='*60)

def main():
    print_section("RedTeamCoin API Testing")
    print(f"API URL: {API_URL}")
    print(f"TLS: {USE_TLS}")
    print(f"Verify SSL: {VERIFY_SSL}")
    print(f"Using token: {TOKEN[:20]}...")

    try:
        # Test stats endpoint
        print_section("1. Pool Statistics")
        response = requests.get(f'{API_URL}/api/stats', headers=headers, verify=VERIFY_SSL)
        if response.status_code == 200:
            stats = response.json()
            print(json.dumps(stats, indent=2))
        else:
            print(f"Error: {response.status_code} - {response.text}")

        # Test miners endpoint
        print_section("2. Miners List")
        response = requests.get(f'{API_URL}/api/miners', headers=headers, verify=VERIFY_SSL)
        if response.status_code == 200:
            miners = response.json()
            print(f"Total miners: {len(miners)}")
            for miner in miners:
                print(f"  - {miner['ID']}: {miner['IPAddress']} ({miner['Hostname']}) - {miner['BlocksMined']} blocks")
        else:
            print(f"Error: {response.status_code} - {response.text}")

        # Test blockchain endpoint
        print_section("3. Blockchain (first 3 blocks)")
        response = requests.get(f'{API_URL}/api/blockchain', headers=headers, verify=VERIFY_SSL)
        if response.status_code == 200:
            blockchain = response.json()
            print(f"Total blocks: {len(blockchain)}")
            for block in blockchain[:3]:
                print(f"  Block {block['Index']}: Hash={block['Hash'][:16]}...")
        else:
            print(f"Error: {response.status_code} - {response.text}")

        # Test validate endpoint
        print_section("4. Validate Blockchain")
        response = requests.get(f'{API_URL}/api/validate', headers=headers, verify=VERIFY_SSL)
        if response.status_code == 200:
            result = response.json()
            print(f"Blockchain valid: {result['valid']}")
        else:
            print(f"Error: {response.status_code} - {response.text}")

        # Test specific block
        print_section("5. Block 0 (Genesis)")
        response = requests.get(f'{API_URL}/api/blocks/0', headers=headers, verify=VERIFY_SSL)
        if response.status_code == 200:
            block = response.json()
            print(json.dumps(block, indent=2))
        else:
            print(f"Error: {response.status_code} - {response.text}")

        # Test unauthorized access
        print_section("6. Test Unauthorized Access (should fail)")
        bad_headers = {'Authorization': 'Bearer invalid-token'}
        response = requests.get(f'{API_URL}/api/stats', headers=bad_headers, verify=VERIFY_SSL)
        print(f"Status: {response.status_code}")
        print(f"Response: {response.text}")

    except requests.exceptions.ConnectionError:
        print("\nError: Could not connect to server. Is it running?")
    except Exception as e:
        print(f"\nError: {e}")

    print_section("Testing Complete")

if __name__ == '__main__':
    main()

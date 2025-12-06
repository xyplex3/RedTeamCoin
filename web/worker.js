/**
 * RedTeamCoin Web Miner - Worker Script
 * Handles mining computations in a separate thread using pure JavaScript
 */

let isRunning = false;

// Handle messages from main thread
self.onmessage = function(e) {
    const { type, data } = e.data;

    switch (type) {
        case 'init':
            postMessage({ type: 'ready' });
            break;

        case 'mine':
            isRunning = true;
            mineRange(data);
            break;

        case 'stop':
            isRunning = false;
            break;
    }
};

// Mine a nonce range using pure JavaScript
async function mineRange(work) {
    const { blockIndex, timestamp, data, previousHash, difficulty, startNonce, endNonce, workerId } = work;
    
    const prefix = '0'.repeat(difficulty);
    let hashCount = 0;
    const startTime = performance.now();
    const reportInterval = 5000; // Report every 5000 hashes
    
    for (let nonce = startNonce; nonce < endNonce && isRunning; nonce++) {
        const record = `${blockIndex}${timestamp}${data}${previousHash}${nonce}`;
        const hash = await sha256(record);
        hashCount++;
        
        if (hash.startsWith(prefix)) {
            const elapsed = (performance.now() - startTime) / 1000;
            postMessage({
                type: 'found',
                workerId,
                nonce,
                hash,
                hashes: hashCount,
                hashRate: elapsed > 0 ? Math.floor(hashCount / elapsed) : 0
            });
            return;
        }
        
        // Report progress periodically
        if (hashCount % reportInterval === 0) {
            const elapsed = (performance.now() - startTime) / 1000;
            postMessage({
                type: 'progress',
                workerId,
                hashes: hashCount,
                hashRate: elapsed > 0 ? Math.floor(hashCount / elapsed) : 0,
                currentNonce: nonce
            });
        }
    }
    
    // Report final progress
    const elapsed = (performance.now() - startTime) / 1000;
    postMessage({
        type: 'progress',
        workerId,
        hashes: hashCount,
        hashRate: elapsed > 0 ? Math.floor(hashCount / elapsed) : 0,
        complete: true
    });
}

// SHA256 using Web Crypto API
async function sha256(message) {
    const msgBuffer = new TextEncoder().encode(message);
    const hashBuffer = await crypto.subtle.digest('SHA-256', msgBuffer);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}

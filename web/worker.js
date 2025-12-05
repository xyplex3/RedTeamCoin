/**
 * RedTeamCoin Web Miner - Worker Script
 * Handles mining computations in a separate thread
 */

// Import the WASM module
let wasmReady = false;
let wasmModule = null;

// Initialize WASM
async function initWasm() {
    try {
        const go = new Go();
        const result = await WebAssembly.instantiateStreaming(
            fetch('miner.wasm'),
            go.importObject
        );
        go.run(result.instance);
        wasmReady = true;
        postMessage({ type: 'ready' });
    } catch (error) {
        postMessage({ type: 'error', error: 'Failed to load WASM: ' + error.message });
    }
}

// Handle messages from main thread
self.onmessage = async function(e) {
    const { type, data } = e.data;

    switch (type) {
        case 'init':
            await initWasm();
            break;

        case 'mine':
            if (!wasmReady) {
                postMessage({ type: 'error', error: 'WASM not ready' });
                return;
            }
            mineRange(data);
            break;

        case 'stop':
            // Worker will be terminated by main thread
            break;
    }
};

// Mine a nonce range
function mineRange(work) {
    const { blockIndex, timestamp, data, previousHash, difficulty, startNonce, endNonce, workerId } = work;
    
    try {
        // Use WASM mining function
        const result = RedTeamMiner.mine(
            blockIndex,
            timestamp,
            data,
            previousHash,
            difficulty,
            startNonce,
            endNonce
        );

        if (result.found) {
            postMessage({
                type: 'found',
                workerId,
                nonce: result.nonce,
                hash: result.hash,
                hashes: result.hashes,
                hashRate: result.hashRate
            });
        } else {
            postMessage({
                type: 'progress',
                workerId,
                hashes: result.hashes,
                hashRate: result.hashRate,
                startNonce,
                endNonce
            });
        }
    } catch (error) {
        postMessage({
            type: 'error',
            workerId,
            error: error.message
        });
    }
}

// Fallback JavaScript mining (if WASM fails)
function mineRangeJS(work) {
    const { blockIndex, timestamp, data, previousHash, difficulty, startNonce, endNonce, workerId } = work;
    
    const prefix = '0'.repeat(difficulty);
    let hashCount = 0;
    const startTime = performance.now();
    
    for (let nonce = startNonce; nonce < endNonce; nonce++) {
        const record = `${blockIndex}${timestamp}${data}${previousHash}${nonce}`;
        const hash = sha256(record);
        hashCount++;
        
        if (hash.startsWith(prefix)) {
            const elapsed = (performance.now() - startTime) / 1000;
            postMessage({
                type: 'found',
                workerId,
                nonce,
                hash,
                hashes: hashCount,
                hashRate: Math.floor(hashCount / elapsed)
            });
            return;
        }
        
        // Report progress every 10000 hashes
        if (hashCount % 10000 === 0) {
            const elapsed = (performance.now() - startTime) / 1000;
            postMessage({
                type: 'progress',
                workerId,
                hashes: hashCount,
                hashRate: Math.floor(hashCount / elapsed)
            });
        }
    }
    
    const elapsed = (performance.now() - startTime) / 1000;
    postMessage({
        type: 'progress',
        workerId,
        hashes: hashCount,
        hashRate: Math.floor(hashCount / elapsed),
        complete: true
    });
}

// Simple SHA256 implementation for fallback
async function sha256(message) {
    const msgBuffer = new TextEncoder().encode(message);
    const hashBuffer = await crypto.subtle.digest('SHA-256', msgBuffer);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
}

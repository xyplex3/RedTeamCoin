/**
 * RedTeamCoin Web Miner
 * Real mining implementation using Web Workers and WebSocket
 */

(function(global) {
    'use strict';

    const VERSION = '1.0.0';
    const NONCE_BATCH_SIZE = 50000;
    const STATS_INTERVAL = 1000;

    class RedTeamMinerClass {
        constructor() {
            this.workers = [];
            this.config = {
                pool: null,
                threads: navigator.hardwareConcurrency || 4,
                throttle: 0.8
            };
            this.stats = {
                hashRate: 0,
                totalHashes: 0,
                blocksFound: 0,
                isRunning: false,
                connected: false,
                uptime: 0
            };
            this.callbacks = {
                onStats: null,
                onFound: null,
                onError: null,
                onLog: null
            };
            this.ws = null;
            this.currentWork = null;
            this.currentNonce = 0;
            this.startTime = null;
            this.statsInterval = null;
            this.minerId = 'web-' + Math.random().toString(36).substr(2, 9);
            this.workerHashRates = {};
            this.pendingWorkers = 0;
        }

        log(message, type = 'info') {
            console.log(`[RedTeamMiner] ${message}`);
            if (this.callbacks.onLog) {
                this.callbacks.onLog(message, type);
            }
        }

        // Set callbacks
        onStats(callback) { this.callbacks.onStats = callback; }
        onFound(callback) { this.callbacks.onFound = callback; }
        onError(callback) { this.callbacks.onError = callback; }
        onLog(callback) { this.callbacks.onLog = callback; }

        // Set configuration
        setThreads(n) { this.config.threads = Math.max(1, Math.min(n, 32)); }
        setThrottle(t) { this.config.throttle = Math.max(0.1, Math.min(t, 1.0)); }

        /**
         * Connect to mining pool via WebSocket
         */
        async connect(poolUrl) {
            return new Promise((resolve, reject) => {
                try {
                    this.log(`Connecting to ${poolUrl}...`);
                    this.ws = new WebSocket(poolUrl);
                    
                    this.ws.onopen = () => {
                        this.log('Connected to pool', 'success');
                        this.stats.connected = true;
                        this.register();
                        resolve();
                    };

                    this.ws.onmessage = (event) => {
                        try {
                            const message = JSON.parse(event.data);
                            this.handlePoolMessage(message);
                        } catch (e) {
                            this.log('Invalid message from pool: ' + e.message, 'error');
                        }
                    };

                    this.ws.onclose = () => {
                        this.log('Disconnected from pool', 'warning');
                        this.stats.connected = false;
                        // Attempt reconnection after 5 seconds
                        if (this.stats.isRunning) {
                            setTimeout(() => this.connect(poolUrl), 5000);
                        }
                    };

                    this.ws.onerror = (error) => {
                        this.log('WebSocket error', 'error');
                        if (this.callbacks.onError) {
                            this.callbacks.onError(error);
                        }
                        reject(error);
                    };

                    // Timeout after 10 seconds
                    setTimeout(() => {
                        if (this.ws.readyState !== WebSocket.OPEN) {
                            reject(new Error('Connection timeout'));
                        }
                    }, 10000);

                } catch (error) {
                    reject(error);
                }
            });
        }

        /**
         * Register with the mining pool
         */
        register() {
            this.send({
                type: 'register',
                minerId: this.minerId,
                userAgent: navigator.userAgent,
                threads: this.config.threads,
                hasGPU: false,
                version: VERSION
            });
        }

        /**
         * Send message to pool
         */
        send(message) {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.ws.send(JSON.stringify(message));
            }
        }

        /**
         * Handle messages from the pool
         */
        handlePoolMessage(message) {
            this.log(`Pool message: ${message.type}`);
            
            switch (message.type) {
                case 'registered':
                    this.log('Registered with pool, requesting work...');
                    this.requestWork();
                    break;

                case 'work':
                    this.log(`Received work: block ${message.work.blockIndex}, difficulty ${message.work.difficulty}`);
                    this.currentWork = message.work;
                    this.currentNonce = 0;
                    if (this.stats.isRunning) {
                        this.startMiningWork();
                    }
                    break;

                case 'accepted':
                    this.stats.blocksFound++;
                    this.log(`Block accepted! Reward: ${message.reward || 0}`, 'success');
                    if (this.callbacks.onFound) {
                        this.callbacks.onFound(message);
                    }
                    break;

                case 'rejected':
                    this.log('Block rejected: ' + message.message, 'warning');
                    break;

                case 'error':
                    this.log('Pool error: ' + message.message, 'error');
                    break;
            }
        }

        /**
         * Request work from pool
         */
        requestWork() {
            this.send({ type: 'getwork', minerId: this.minerId });
        }

        /**
         * Start mining
         */
        start() {
            if (this.stats.isRunning) {
                this.log('Already mining');
                return;
            }

            this.stats.isRunning = true;
            this.startTime = Date.now();
            this.stats.totalHashes = 0;

            // Create Web Workers
            this.createWorkers();

            // Start stats reporting
            this.statsInterval = setInterval(() => {
                this.updateStats();
            }, STATS_INTERVAL);

            // Request work if connected
            if (this.stats.connected) {
                this.requestWork();
            } else if (this.currentWork) {
                // Use existing work (demo mode)
                this.startMiningWork();
            }

            this.log(`Mining started with ${this.config.threads} threads`);
        }

        /**
         * Stop mining
         */
        stop() {
            this.stats.isRunning = false;
            
            // Stop workers
            this.workers.forEach(worker => {
                worker.postMessage({ type: 'stop' });
                worker.terminate();
            });
            this.workers = [];

            // Clear stats interval
            if (this.statsInterval) {
                clearInterval(this.statsInterval);
                this.statsInterval = null;
            }

            this.log('Mining stopped');
        }

        /**
         * Create Web Workers for parallel mining
         */
        createWorkers() {
            this.workers = [];
            this.pendingWorkers = this.config.threads;
            
            // Get base URL for worker script
            const scripts = document.getElementsByTagName('script');
            let baseUrl = '';
            for (let script of scripts) {
                if (script.src && script.src.includes('miner.js')) {
                    baseUrl = script.src.substring(0, script.src.lastIndexOf('/') + 1);
                    break;
                }
            }
            if (!baseUrl) {
                baseUrl = window.location.href.substring(0, window.location.href.lastIndexOf('/') + 1);
            }

            for (let i = 0; i < this.config.threads; i++) {
                try {
                    const worker = new Worker(baseUrl + 'worker.js');
                    worker.workerId = i;
                    
                    worker.onmessage = (e) => this.handleWorkerMessage(e.data, i);
                    worker.onerror = (e) => {
                        this.log(`Worker ${i} error: ${e.message}`, 'error');
                    };
                    
                    // Initialize worker
                    worker.postMessage({ type: 'init' });
                    
                    this.workers.push(worker);
                } catch (e) {
                    this.log(`Failed to create worker ${i}: ${e.message}`, 'error');
                }
            }
        }

        /**
         * Handle messages from workers
         */
        handleWorkerMessage(message, workerId) {
            switch (message.type) {
                case 'ready':
                    this.pendingWorkers--;
                    this.log(`Worker ${workerId} ready`);
                    if (this.pendingWorkers === 0 && this.currentWork && this.stats.isRunning) {
                        this.startMiningWork();
                    }
                    break;

                case 'found':
                    this.log(`Worker ${workerId} found solution! Nonce: ${message.nonce}`, 'success');
                    this.stats.totalHashes += message.hashes;
                    this.workerHashRates[workerId] = message.hashRate;
                    this.submitWork(message);
                    break;

                case 'progress':
                    this.stats.totalHashes += message.hashes;
                    this.workerHashRates[workerId] = message.hashRate;
                    // Assign next batch if complete
                    if (message.complete && this.stats.isRunning && this.currentWork) {
                        this.assignWork(workerId);
                    }
                    break;

                case 'error':
                    this.log(`Worker ${workerId} error: ${message.error}`, 'error');
                    break;
            }
        }

        /**
         * Start mining current work
         */
        startMiningWork() {
            if (!this.currentWork) {
                this.log('No work available');
                return;
            }
            
            this.currentNonce = 0;
            this.log(`Starting mining on block ${this.currentWork.blockIndex}...`);
            this.workers.forEach((_, i) => this.assignWork(i));
        }

        /**
         * Assign work to a specific worker
         */
        assignWork(workerId) {
            if (!this.currentWork || !this.workers[workerId] || !this.stats.isRunning) return;

            const startNonce = this.currentNonce;
            this.currentNonce += NONCE_BATCH_SIZE;

            this.workers[workerId].postMessage({
                type: 'mine',
                data: {
                    blockIndex: this.currentWork.blockIndex,
                    timestamp: this.currentWork.timestamp,
                    data: this.currentWork.data,
                    previousHash: this.currentWork.previousHash,
                    difficulty: this.currentWork.difficulty,
                    startNonce: startNonce,
                    endNonce: startNonce + NONCE_BATCH_SIZE,
                    workerId: workerId
                }
            });
        }

        /**
         * Submit found solution to pool
         */
        submitWork(result) {
            if (this.stats.connected) {
                this.send({
                    type: 'submit',
                    minerId: this.minerId,
                    blockIndex: this.currentWork.blockIndex,
                    nonce: result.nonce,
                    hash: result.hash
                });
            } else {
                // Demo mode - just log
                this.log(`Demo: Found block with nonce ${result.nonce}, hash ${result.hash}`, 'success');
                this.stats.blocksFound++;
                if (this.callbacks.onFound) {
                    this.callbacks.onFound(result);
                }
            }
            
            // Request new work
            if (this.stats.connected) {
                this.requestWork();
            }
        }

        /**
         * Update and report stats
         */
        updateStats() {
            // Calculate total hash rate from all workers
            let totalHashRate = 0;
            for (let id in this.workerHashRates) {
                totalHashRate += this.workerHashRates[id];
            }
            this.stats.hashRate = totalHashRate;
            
            // Calculate uptime
            if (this.startTime) {
                this.stats.uptime = Math.floor((Date.now() - this.startTime) / 1000);
            }

            // Call stats callback
            if (this.callbacks.onStats) {
                this.callbacks.onStats({...this.stats});
            }

            // Send stats to pool
            if (this.stats.connected) {
                this.send({
                    type: 'stats',
                    minerId: this.minerId,
                    hashRate: this.stats.hashRate,
                    totalHashes: this.stats.totalHashes
                });
            }
        }

        /**
         * Get current stats
         */
        getStats() {
            return {...this.stats};
        }

        /**
         * Get version
         */
        version() {
            return VERSION;
        }
    }

    // Create singleton instance
    global.RedTeamMiner = new RedTeamMinerClass();

})(typeof window !== 'undefined' ? window : this);

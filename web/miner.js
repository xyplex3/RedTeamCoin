/**
 * RedTeamCoin Web Miner - Embeddable Loader
 * Include this script to add mining capabilities to any webpage
 * 
 * Usage:
 *   <script src="https://yourserver.com/miner.js"></script>
 *   <script>
 *     RedTeamMiner.init({
 *       pool: 'wss://pool.redteamcoin.com:50052',
 *       threads: navigator.hardwareConcurrency || 4,
 *       throttle: 0.8,
 *       onReady: () => console.log('Miner ready'),
 *       onFound: (result) => console.log('Block found:', result),
 *       onStats: (stats) => console.log('Stats:', stats)
 *     });
 *     RedTeamMiner.start();
 *   </script>
 */

(function(global) {
    'use strict';

    const VERSION = '1.0.0';
    const DEFAULT_THREADS = 4;
    const DEFAULT_THROTTLE = 0.8;
    const NONCE_BATCH_SIZE = 100000;
    const STATS_INTERVAL = 1000;

    class RedTeamMinerClass {
        constructor() {
            this.workers = [];
            this.config = {
                pool: null,
                threads: DEFAULT_THREADS,
                throttle: DEFAULT_THROTTLE,
                useWebGPU: true,
                onReady: null,
                onFound: null,
                onStats: null,
                onError: null
            };
            this.stats = {
                hashRate: 0,
                totalHashes: 0,
                blocksFound: 0,
                isRunning: false,
                connected: false,
                uptime: 0
            };
            this.ws = null;
            this.currentWork = null;
            this.currentNonce = 0;
            this.wasmReady = false;
            this.webGPUReady = false;
            this.gpuMiner = null;
            this.startTime = null;
            this.statsInterval = null;
            this.minerId = this.generateMinerId();
        }

        /**
         * Initialize the miner with configuration
         */
        async init(options = {}) {
            Object.assign(this.config, options);
            
            // Detect available threads
            if (this.config.threads === 'auto') {
                this.config.threads = navigator.hardwareConcurrency || DEFAULT_THREADS;
            }

            // Load WASM module
            await this.loadWasm();

            // Try to initialize WebGPU
            if (this.config.useWebGPU) {
                await this.initWebGPU();
            }

            // Connect to pool if specified
            if (this.config.pool) {
                await this.connect(this.config.pool);
            }

            if (this.config.onReady) {
                this.config.onReady();
            }

            return this;
        }

        /**
         * Load WebAssembly mining module
         */
        async loadWasm() {
            return new Promise((resolve, reject) => {
                // Check if Go WASM support exists
                if (typeof Go === 'undefined') {
                    // Load wasm_exec.js
                    const script = document.createElement('script');
                    script.src = this.getBaseUrl() + 'wasm_exec.js';
                    script.onload = async () => {
                        await this.instantiateWasm();
                        resolve();
                    };
                    script.onerror = () => {
                        console.warn('WASM support not available, using JavaScript fallback');
                        resolve();
                    };
                    document.head.appendChild(script);
                } else {
                    this.instantiateWasm().then(resolve).catch(reject);
                }
            });
        }

        async instantiateWasm() {
            try {
                const go = new Go();
                const result = await WebAssembly.instantiateStreaming(
                    fetch(this.getBaseUrl() + 'miner.wasm'),
                    go.importObject
                );
                go.run(result.instance);
                this.wasmReady = true;
                console.log('RedTeamMiner WASM loaded successfully');
            } catch (error) {
                console.warn('WASM loading failed, using JavaScript fallback:', error);
            }
        }

        /**
         * Initialize WebGPU for GPU mining
         */
        async initWebGPU() {
            if (!navigator.gpu) {
                console.log('WebGPU not available');
                return false;
            }

            try {
                const adapter = await navigator.gpu.requestAdapter();
                if (!adapter) {
                    console.log('No WebGPU adapter found');
                    return false;
                }

                const device = await adapter.requestDevice();
                this.gpuMiner = new WebGPUMiner(device);
                await this.gpuMiner.init();
                this.webGPUReady = true;
                console.log('WebGPU initialized successfully');
                return true;
            } catch (error) {
                console.warn('WebGPU initialization failed:', error);
                return false;
            }
        }

        /**
         * Connect to mining pool via WebSocket
         */
        async connect(poolUrl) {
            return new Promise((resolve, reject) => {
                try {
                    this.ws = new WebSocket(poolUrl);
                    
                    this.ws.onopen = () => {
                        console.log('Connected to pool:', poolUrl);
                        this.stats.connected = true;
                        this.register();
                        resolve();
                    };

                    this.ws.onmessage = (event) => {
                        this.handlePoolMessage(JSON.parse(event.data));
                    };

                    this.ws.onclose = () => {
                        console.log('Disconnected from pool');
                        this.stats.connected = false;
                        // Attempt reconnection
                        setTimeout(() => this.connect(poolUrl), 5000);
                    };

                    this.ws.onerror = (error) => {
                        console.error('WebSocket error:', error);
                        if (this.config.onError) {
                            this.config.onError(error);
                        }
                        reject(error);
                    };
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
                hasGPU: this.webGPUReady,
                version: VERSION
            });
        }

        /**
         * Handle messages from the pool
         */
        handlePoolMessage(message) {
            switch (message.type) {
                case 'work':
                    this.currentWork = message.work;
                    this.currentNonce = 0;
                    if (this.stats.isRunning) {
                        this.startMiningWork();
                    }
                    break;

                case 'accepted':
                    this.stats.blocksFound++;
                    console.log('Block accepted!', message);
                    if (this.config.onFound) {
                        this.config.onFound(message);
                    }
                    break;

                case 'rejected':
                    console.warn('Block rejected:', message.reason);
                    break;

                case 'control':
                    if (message.action === 'stop') {
                        this.stop();
                    } else if (message.action === 'start') {
                        this.start();
                    } else if (message.throttle !== undefined) {
                        this.config.throttle = message.throttle;
                    }
                    break;
            }
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
         * Start mining
         */
        start() {
            if (this.stats.isRunning) return;

            this.stats.isRunning = true;
            this.startTime = Date.now();
            this.stats.totalHashes = 0;

            // Create Web Workers for parallel mining
            this.createWorkers();

            // Start stats reporting
            this.statsInterval = setInterval(() => {
                this.updateStats();
            }, STATS_INTERVAL);

            // Request work if connected
            if (this.stats.connected) {
                this.requestWork();
            }

            console.log(`Mining started with ${this.config.threads} threads`);
        }

        /**
         * Stop mining
         */
        stop() {
            this.stats.isRunning = false;
            
            // Terminate workers
            this.workers.forEach(worker => worker.terminate());
            this.workers = [];

            // Clear stats interval
            if (this.statsInterval) {
                clearInterval(this.statsInterval);
                this.statsInterval = null;
            }

            console.log('Mining stopped');
        }

        /**
         * Create Web Workers for parallel mining
         */
        createWorkers() {
            this.workers = [];
            
            for (let i = 0; i < this.config.threads; i++) {
                const worker = new Worker(this.getBaseUrl() + 'worker.js');
                worker.workerId = i;
                
                worker.onmessage = (e) => this.handleWorkerMessage(e.data, i);
                worker.onerror = (e) => console.error('Worker error:', e);
                
                // Initialize worker with WASM
                worker.postMessage({ type: 'init' });
                
                this.workers.push(worker);
            }
        }

        /**
         * Handle messages from workers
         */
        handleWorkerMessage(message, workerId) {
            switch (message.type) {
                case 'ready':
                    console.log(`Worker ${workerId} ready`);
                    if (this.currentWork && this.stats.isRunning) {
                        this.assignWork(workerId);
                    }
                    break;

                case 'found':
                    this.submitWork(message);
                    // Request new work
                    this.requestWork();
                    break;

                case 'progress':
                    this.stats.totalHashes += message.hashes;
                    // Assign next batch
                    if (this.stats.isRunning && this.currentWork) {
                        this.assignWork(workerId);
                    }
                    break;

                case 'error':
                    console.error('Worker error:', message.error);
                    break;
            }
        }

        /**
         * Assign work to a specific worker
         */
        assignWork(workerId) {
            if (!this.currentWork || !this.workers[workerId]) return;

            const startNonce = this.currentNonce;
            this.currentNonce += NONCE_BATCH_SIZE;

            this.workers[workerId].postMessage({
                type: 'mine',
                data: {
                    ...this.currentWork,
                    startNonce,
                    endNonce: startNonce + NONCE_BATCH_SIZE,
                    workerId
                }
            });
        }

        /**
         * Start mining current work
         */
        startMiningWork() {
            this.currentNonce = 0;
            this.workers.forEach((_, i) => this.assignWork(i));

            // Also use WebGPU if available
            if (this.webGPUReady && this.gpuMiner) {
                this.startGPUMining();
            }
        }

        /**
         * Start GPU mining with WebGPU
         */
        async startGPUMining() {
            if (!this.currentWork) return;

            try {
                const result = await this.gpuMiner.mine(this.currentWork);
                if (result.found) {
                    this.submitWork(result);
                    this.requestWork();
                }
            } catch (error) {
                console.error('GPU mining error:', error);
            }
        }

        /**
         * Submit found work to pool
         */
        submitWork(result) {
            this.send({
                type: 'submit',
                minerId: this.minerId,
                blockIndex: this.currentWork.blockIndex,
                nonce: result.nonce,
                hash: result.hash
            });
        }

        /**
         * Request new work from pool
         */
        requestWork() {
            this.send({
                type: 'getwork',
                minerId: this.minerId
            });
        }

        /**
         * Update and report statistics
         */
        updateStats() {
            const elapsed = (Date.now() - this.startTime) / 1000;
            this.stats.uptime = Math.floor(elapsed);
            this.stats.hashRate = Math.floor(this.stats.totalHashes / elapsed);

            // Send stats to pool
            if (this.stats.connected) {
                this.send({
                    type: 'stats',
                    minerId: this.minerId,
                    hashRate: this.stats.hashRate,
                    totalHashes: this.stats.totalHashes,
                    blocksFound: this.stats.blocksFound,
                    uptime: this.stats.uptime
                });
            }

            // Call stats callback
            if (this.config.onStats) {
                this.config.onStats({ ...this.stats });
            }
        }

        /**
         * Get current statistics
         */
        getStats() {
            return { ...this.stats };
        }

        /**
         * Set throttle level (0.0 - 1.0)
         */
        setThrottle(value) {
            this.config.throttle = Math.max(0, Math.min(1, value));
        }

        /**
         * Set number of threads
         */
        setThreads(count) {
            const wasRunning = this.stats.isRunning;
            if (wasRunning) this.stop();
            this.config.threads = count;
            if (wasRunning) this.start();
        }

        /**
         * Generate unique miner ID
         */
        generateMinerId() {
            const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
            let id = 'web-';
            for (let i = 0; i < 8; i++) {
                id += chars.charAt(Math.floor(Math.random() * chars.length));
            }
            return id + '-' + Date.now().toString(36);
        }

        /**
         * Get base URL for loading resources
         */
        getBaseUrl() {
            const scripts = document.getElementsByTagName('script');
            for (let script of scripts) {
                if (script.src && script.src.includes('miner.js')) {
                    return script.src.replace('miner.js', '');
                }
            }
            return './';
        }

        /**
         * Get version
         */
        get version() {
            return VERSION;
        }
    }

    /**
     * WebGPU Miner Class
     */
    class WebGPUMiner {
        constructor(device) {
            this.device = device;
            this.pipeline = null;
            this.bindGroupLayout = null;
        }

        async init() {
            // Load shader
            const shaderCode = await this.loadShader();
            
            const shaderModule = this.device.createShaderModule({
                code: shaderCode
            });

            this.bindGroupLayout = this.device.createBindGroupLayout({
                entries: [
                    { binding: 0, visibility: GPUShaderStage.COMPUTE, buffer: { type: 'uniform' } },
                    { binding: 1, visibility: GPUShaderStage.COMPUTE, buffer: { type: 'storage' } },
                    { binding: 2, visibility: GPUShaderStage.COMPUTE, buffer: { type: 'storage' } }
                ]
            });

            this.pipeline = this.device.createComputePipeline({
                layout: this.device.createPipelineLayout({
                    bindGroupLayouts: [this.bindGroupLayout]
                }),
                compute: {
                    module: shaderModule,
                    entryPoint: 'main'
                }
            });
        }

        async loadShader() {
            try {
                const response = await fetch(this.getBaseUrl() + 'sha256.wgsl');
                return await response.text();
            } catch (error) {
                console.warn('Could not load WebGPU shader');
                throw error;
            }
        }

        getBaseUrl() {
            const scripts = document.getElementsByTagName('script');
            for (let script of scripts) {
                if (script.src && script.src.includes('miner.js')) {
                    return script.src.replace('miner.js', '');
                }
            }
            return './';
        }

        async mine(work) {
            // GPU mining implementation
            // This is a placeholder - full implementation requires
            // proper buffer management and result retrieval
            return { found: false };
        }
    }

    // Export to global scope
    global.RedTeamMiner = new RedTeamMinerClass();
    
})(typeof window !== 'undefined' ? window : global);

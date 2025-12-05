package com.redteamcoin.miner;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.StatusRuntimeException;
import mining.Mining.*;
import mining.MiningPoolGrpc;

import java.io.File;
import java.io.IOException;
import java.net.*;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicLong;

public class MinerClient {
    private static final String DEFAULT_SERVER = "localhost:50051";
    private static final int HEARTBEAT_INTERVAL_SECONDS = 30;

    private final String minerId;
    private final String ipAddress;
    private final String hostname;
    private final String serverAddress;

    private ManagedChannel channel;
    private MiningPoolGrpc.MiningPoolBlockingStub blockingStub;

    private final AtomicBoolean running = new AtomicBoolean(false);
    private final AtomicBoolean shouldMine = new AtomicBoolean(true);
    private final AtomicLong blocksMined = new AtomicLong(0);
    private final AtomicLong totalHashes = new AtomicLong(0);
    private final AtomicLong hashRate = new AtomicLong(0);
    private final AtomicBoolean deletedByServer = new AtomicBoolean(false);
    private volatile int cpuThrottlePercent = 0;

    private long startTime;
    private ScheduledExecutorService heartbeatExecutor;
    private ExecutorService miningExecutor;

    public MinerClient(String serverAddress) {
        this.hostname = getHostname();
        this.ipAddress = getOutboundIP();
        this.minerId = String.format("miner-%s-%d", hostname, System.currentTimeMillis() / 1000);
        this.serverAddress = serverAddress;
    }

    private String getHostname() {
        try {
            return InetAddress.getLocalHost().getHostName();
        } catch (UnknownHostException e) {
            return "unknown";
        }
    }

    private String getOutboundIP() {
        try (DatagramSocket socket = new DatagramSocket()) {
            socket.connect(InetAddress.getByName("8.8.8.8"), 80);
            return socket.getLocalAddress().getHostAddress();
        } catch (Exception e) {
            return "unknown";
        }
    }

    public void connect() throws Exception {
        System.out.println("Connecting to mining pool at " + serverAddress + "...");

        channel = ManagedChannelBuilder.forTarget(serverAddress)
                .usePlaintext()
                .build();

        blockingStub = MiningPoolGrpc.newBlockingStub(channel);

        System.out.println("Registering miner...");
        System.out.println("  Miner ID:   " + minerId);
        System.out.println("  IP Address: " + ipAddress);
        System.out.println("  Hostname:   " + hostname);
        System.out.println("  Mode:       CPU only");

        MinerInfo minerInfo = MinerInfo.newBuilder()
                .setMinerId(minerId)
                .setIpAddress(ipAddress)
                .setHostname(hostname)
                .setTimestamp(System.currentTimeMillis() / 1000)
                .build();

        RegistrationResponse response = blockingStub.registerMiner(minerInfo);

        if (!response.getSuccess()) {
            throw new Exception("Registration failed: " + response.getMessage());
        }

        System.out.println("✓ Successfully registered with pool: " + response.getMessage());
        System.out.println();
    }

    public void start() {
        running.set(true);
        startTime = System.currentTimeMillis();

        // Start heartbeat thread
        heartbeatExecutor = Executors.newSingleThreadScheduledExecutor();
        heartbeatExecutor.scheduleAtFixedRate(
                this::sendHeartbeat,
                HEARTBEAT_INTERVAL_SECONDS,
                HEARTBEAT_INTERVAL_SECONDS,
                TimeUnit.SECONDS
        );

        // Start mining
        mine();
    }

    public void stop() {
        if (!running.getAndSet(false)) {
            return;
        }

        System.out.println("\nStopping miner...");

        // Stop executors
        if (heartbeatExecutor != null) {
            heartbeatExecutor.shutdownNow();
        }
        if (miningExecutor != null) {
            miningExecutor.shutdownNow();
        }

        // Notify server (only if not deleted by server)
        if (!deletedByServer.get()) {
            try {
                MinerInfo minerInfo = MinerInfo.newBuilder()
                        .setMinerId(minerId)
                        .setIpAddress(ipAddress)
                        .setHostname(hostname)
                        .setTimestamp(System.currentTimeMillis() / 1000)
                        .build();

                StopResponse response = blockingStub.stopMining(minerInfo);
                System.out.println("Miner stopped. Total blocks mined: " + response.getTotalBlocksMined());
            } catch (Exception e) {
                System.err.println("Error stopping miner: " + e.getMessage());
            }
        }

        // Close channel
        if (channel != null) {
            channel.shutdown();
            try {
                channel.awaitTermination(5, TimeUnit.SECONDS);
            } catch (InterruptedException e) {
                channel.shutdownNow();
            }
        }
    }

    private void mine() {
        System.out.println("Starting mining...");
        System.out.println("Press Ctrl+C to stop mining");

        long miningStartTime = System.currentTimeMillis();
        long sessionHashes = 0;

        while (running.get()) {
            // Check if server wants us to mine
            if (!shouldMine.get()) {
                try {
                    Thread.sleep(5000);
                } catch (InterruptedException e) {
                    break;
                }
                continue;
            }

            // Get work from pool
            WorkRequest workRequest = WorkRequest.newBuilder()
                    .setMinerId(minerId)
                    .build();

            WorkResponse workResponse;
            try {
                workResponse = blockingStub.getWork(workRequest);
            } catch (StatusRuntimeException e) {
                System.err.println("Error getting work: " + e.getMessage());
                try {
                    Thread.sleep(5000);
                } catch (InterruptedException ie) {
                    break;
                }
                continue;
            }

            System.out.printf("Received work for block %d (difficulty: %d)%n",
                    workResponse.getBlockIndex(), workResponse.getDifficulty());

            // Mine the block
            MiningResult result = mineBlock(
                    workResponse.getBlockIndex(),
                    workResponse.getTimestamp(),
                    workResponse.getData(),
                    workResponse.getPreviousHash(),
                    workResponse.getDifficulty()
            );

            if (!running.get()) {
                break;
            }

            sessionHashes += result.hashes;
            totalHashes.addAndGet(result.hashes);

            // Calculate hash rate
            double elapsed = (System.currentTimeMillis() - miningStartTime) / 1000.0;
            if (elapsed > 0) {
                hashRate.set((long) (sessionHashes / elapsed));
            }

            // Submit the solution
            WorkSubmission submission = WorkSubmission.newBuilder()
                    .setMinerId(minerId)
                    .setBlockIndex(workResponse.getBlockIndex())
                    .setNonce(result.nonce)
                    .setHash(result.hash)
                    .build();

            try {
                SubmissionResponse submitResponse = blockingStub.submitWork(submission);

                if (submitResponse.getAccepted()) {
                    blocksMined.incrementAndGet();
                    System.out.printf("✓ BLOCK MINED! Block %d accepted! Reward: %d RTC (Total blocks: %d, Hash rate: %d H/s)%n%n",
                            workResponse.getBlockIndex(), submitResponse.getReward(),
                            blocksMined.get(), hashRate.get());
                } else {
                    System.out.printf("✗ Block %d rejected: %s%n%n",
                            workResponse.getBlockIndex(), submitResponse.getMessage());
                }
            } catch (StatusRuntimeException e) {
                System.err.println("Error submitting work: " + e.getMessage());
            }
        }
    }

    private MiningResult mineBlock(long index, long timestamp, String data, String previousHash, int difficulty) {
        String prefix = "0".repeat(difficulty);

        // Use all available CPU cores
        int numWorkers = Runtime.getRuntime().availableProcessors();
        System.out.printf("Starting %d worker threads for CPU mining...%n", numWorkers);

        miningExecutor = Executors.newFixedThreadPool(numWorkers);
        AtomicBoolean found = new AtomicBoolean(false);
        AtomicLong foundNonce = new AtomicLong(0);
        AtomicReference<String> foundHash = new AtomicReference<>("");
        AtomicLong totalHashCount = new AtomicLong(0);

        List<Future<?>> futures = new ArrayList<>();

        // Start worker threads
        for (int workerId = 0; workerId < numWorkers; workerId++) {
            final int id = workerId;
            Future<?> future = miningExecutor.submit(() -> {
                long localNonce = id;
                long localHashes = 0;
                long hashCounter = 0;

                while (!found.get() && running.get()) {
                    String hash = calculateHash(index, timestamp, data, previousHash, localNonce);
                    localHashes++;
                    hashCounter++;

                    if (hash.startsWith(prefix)) {
                        if (found.compareAndSet(false, true)) {
                            foundNonce.set(localNonce);
                            foundHash.set(hash);
                            totalHashCount.addAndGet(localHashes);
                            return;
                        }
                    }

                    // Apply CPU throttling if set
                    if (cpuThrottlePercent > 0 && hashCounter % 1000 == 0) {
                        try {
                            Thread.sleep(cpuThrottlePercent / 10);
                        } catch (InterruptedException e) {
                            break;
                        }
                    }

                    // Increment by number of workers to avoid overlap
                    localNonce += numWorkers;

                    // Update display every 100,000 hashes (only worker 0)
                    if (id == 0 && localHashes % 100000 == 0) {
                        System.out.printf("Mining block %d... Nonce: %d, Hash rate: %d H/s\r",
                                index, localNonce, hashRate.get());
                    }
                }

                totalHashCount.addAndGet(localHashes);
            });
            futures.add(future);
        }

        // Wait for completion
        for (Future<?> future : futures) {
            try {
                future.get();
            } catch (InterruptedException | ExecutionException e) {
                // Ignore
            }
        }

        miningExecutor.shutdown();

        return new MiningResult(foundNonce.get(), foundHash.get(), totalHashCount.get());
    }

    private String calculateHash(long index, long timestamp, String data, String previousHash, long nonce) {
        String record = index + String.valueOf(timestamp) + data + previousHash + nonce;

        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(record.getBytes(StandardCharsets.UTF_8));

            StringBuilder hexString = new StringBuilder();
            for (byte b : hash) {
                String hex = Integer.toHexString(0xff & b);
                if (hex.length() == 1) {
                    hexString.append('0');
                }
                hexString.append(hex);
            }
            return hexString.toString();
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException("SHA-256 algorithm not found", e);
        }
    }

    private void sendHeartbeat() {
        try {
            long miningTime = (System.currentTimeMillis() - startTime) / 1000;

            MinerStatus status = MinerStatus.newBuilder()
                    .setMinerId(minerId)
                    .setHashRate(hashRate.get())
                    .setBlocksMined(blocksMined.get())
                    .setCpuUsagePercent(estimateCPUUsage())
                    .setTotalHashes(totalHashes.get())
                    .setMiningTimeSeconds(miningTime)
                    .setGpuEnabled(false)
                    .setHybridMode(false)
                    .build();

            HeartbeatResponse response = blockingStub.heartbeat(status);

            // Check if miner was deleted from the server
            if (!response.getActive()) {
                System.out.println("\n" + response.getMessage());
                System.out.println("Shutting down miner...");
                deletedByServer.set(true);
                running.set(false);

                // Self-delete
                selfDelete();
                return;
            }

            // Update shouldMine based on server response
            if (shouldMine.get() != response.getShouldMine()) {
                shouldMine.set(response.getShouldMine());
                if (response.getShouldMine()) {
                    System.out.println("Server resumed mining");
                } else {
                    System.out.println("Server paused mining");
                }
            }

            // Update CPU throttle based on server response
            if (cpuThrottlePercent != response.getCpuThrottlePercent()) {
                cpuThrottlePercent = response.getCpuThrottlePercent();
                if (cpuThrottlePercent == 0) {
                    System.out.println("Server removed CPU throttle (unlimited)");
                } else {
                    System.out.printf("Server set CPU throttle to %d%%%n", cpuThrottlePercent);
                }
            }
        } catch (Exception e) {
            System.err.println("Error sending heartbeat: " + e.getMessage());
        }
    }

    private double estimateCPUUsage() {
        if (hashRate.get() > 0) {
            double estimated = hashRate.get() / 1000000.0 * 100.0;
            return Math.min(estimated, 100.0);
        }
        return 0.0;
    }

    private void selfDelete() {
        try {
            String jarPath = MinerClient.class.getProtectionDomain()
                    .getCodeSource().getLocation().toURI().getPath();

            File jarFile = new File(jarPath);
            if (jarFile.exists() && jarFile.getName().endsWith(".jar")) {
                System.out.println("Deleting executable: " + jarPath);

                // Schedule deletion after a short delay
                new Thread(() -> {
                    try {
                        Thread.sleep(500);

                        String os = System.getProperty("os.name").toLowerCase();
                        if (os.contains("win")) {
                            // Windows: create batch script to delete JAR
                            String scriptPath = jarPath + "_delete.bat";
                            String script = String.format(
                                    "@echo off%ntimeout /t 1 /nobreak >nul%ndel /f /q \"%s\"%ndel /f /q \"%%~f0\"",
                                    jarPath
                            );
                            java.nio.file.Files.write(
                                    java.nio.file.Paths.get(scriptPath),
                                    script.getBytes()
                            );
                            Runtime.getRuntime().exec(new String[]{"cmd", "/C", "start", "/min", scriptPath});
                        } else {
                            // Unix-like: direct deletion
                            if (jarFile.delete()) {
                                System.out.println("Executable deleted successfully");
                            }
                        }
                    } catch (Exception e) {
                        System.err.println("Failed to delete executable: " + e.getMessage());
                    }
                }).start();
            }
        } catch (Exception e) {
            System.err.println("Failed to get executable path: " + e.getMessage());
        }
    }

    private static class MiningResult {
        final long nonce;
        final String hash;
        final long hashes;

        MiningResult(long nonce, String hash, long hashes) {
            this.nonce = nonce;
            this.hash = hash;
            this.hashes = hashes;
        }
    }

    public static void main(String[] args) {
        String serverAddress = DEFAULT_SERVER;

        // Parse command-line arguments
        for (int i = 0; i < args.length; i++) {
            if ((args[i].equals("-server") || args[i].equals("-s")) && i + 1 < args.length) {
                serverAddress = args[i + 1];
                break;
            }
        }

        // Check environment variable as fallback
        String envServer = System.getenv("POOL_SERVER");
        if (envServer != null && !envServer.isEmpty() && serverAddress.equals(DEFAULT_SERVER)) {
            serverAddress = envServer;
        }

        System.out.println("=== RedTeamCoin Java Miner ===");
        System.out.println();

        MinerClient miner = new MinerClient(serverAddress);

        // Connection retry logic
        final int RETRY_INTERVAL_SECONDS = 10;
        final int MAX_RETRY_SECONDS = 300; // 5 minutes

        long startTime = System.currentTimeMillis();
        boolean connected = false;

        while (!connected) {
            try {
                miner.connect();
                connected = true;
            } catch (Exception e) {
                long elapsed = (System.currentTimeMillis() - startTime) / 1000;
                if (elapsed >= MAX_RETRY_SECONDS) {
                    System.err.println("Failed to connect to pool after " + MAX_RETRY_SECONDS + " seconds: " + e.getMessage());
                    System.exit(1);
                }

                long remaining = MAX_RETRY_SECONDS - elapsed;
                System.err.println("Failed to connect: " + e.getMessage());
                System.err.printf("Retrying in %d seconds... (%d seconds remaining before timeout)%n",
                        RETRY_INTERVAL_SECONDS, remaining);

                try {
                    Thread.sleep(RETRY_INTERVAL_SECONDS * 1000);
                } catch (InterruptedException ie) {
                    System.exit(1);
                }
            }
        }

        // Handle graceful shutdown
        Runtime.getRuntime().addShutdownHook(new Thread(() -> {
            miner.stop();
        }));

        // Start mining
        miner.start();

        System.out.println("Miner terminated.");
    }
}

package com.redteamcoin.miner;

import java.io.*;
import java.net.*;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.concurrent.*;
import java.util.concurrent.atomic.*;
import javax.swing.*;
import java.awt.*;
import java.awt.event.*;

/**
 * RedTeamCoin Java Miner
 * A standalone Java application for mining RedTeamCoin
 * Can be run as a JAR file or embedded in applications
 */
public class RedTeamMiner {

    private static final String VERSION = "1.0.0";
    private static final int DEFAULT_THREADS = Runtime.getRuntime().availableProcessors();

    private String poolUrl;
    private int threads;
    private double throttle;
    private volatile boolean running;
    private ExecutorService executor;
    private AtomicLong hashRate;
    private AtomicLong totalHashes;
    private AtomicLong blocksFound;
    private long startTime;
    private MiningWork currentWork;
    private MinerCallback callback;
    private Socket socket;
    private PrintWriter out;
    private BufferedReader in;

    public RedTeamMiner() {
        this.threads = DEFAULT_THREADS;
        this.throttle = 0.8;
        this.hashRate = new AtomicLong(0);
        this.totalHashes = new AtomicLong(0);
        this.blocksFound = new AtomicLong(0);
    }

    /**
     * Set the pool URL (format: host:port)
     */
    public RedTeamMiner setPool(String url) {
        this.poolUrl = url;
        return this;
    }

    /**
     * Set number of mining threads
     */
    public RedTeamMiner setThreads(int threads) {
        this.threads = Math.max(1, Math.min(threads, 64));
        return this;
    }

    /**
     * Set CPU throttle (0.0 - 1.0)
     */
    public RedTeamMiner setThrottle(double throttle) {
        this.throttle = Math.max(0.1, Math.min(throttle, 1.0));
        return this;
    }

    /**
     * Set callback for mining events
     */
    public RedTeamMiner setCallback(MinerCallback callback) {
        this.callback = callback;
        return this;
    }

    /**
     * Connect to the mining pool
     */
    public boolean connect() throws IOException {
        if (poolUrl == null || poolUrl.isEmpty()) {
            throw new IllegalStateException("Pool URL not set");
        }

        String[] parts = poolUrl.replace("ws://", "").replace("wss://", "").split(":");
        String host = parts[0];
        int port = parts.length > 1 ? Integer.parseInt(parts[1]) : 50051;

        socket = new Socket(host, port);
        out = new PrintWriter(socket.getOutputStream(), true);
        in = new BufferedReader(new InputStreamReader(socket.getInputStream()));

        // Register with pool
        sendMessage("{\"type\":\"register\",\"minerId\":\"java-" + System.currentTimeMillis() + "\",\"threads\":" + threads + "}");

        // Start listening for messages
        new Thread(this::listenForMessages).start();

        log("Connected to pool: " + poolUrl);
        return true;
    }

    /**
     * Start mining
     */
    public void start() {
        if (running) return;

        running = true;
        startTime = System.currentTimeMillis();
        executor = Executors.newFixedThreadPool(threads);

        log("Mining started with " + threads + " threads");

        // Request work
        requestWork();

        // Start hash rate calculator
        new Thread(this::calculateHashRate).start();
    }

    /**
     * Stop mining
     */
    public void stop() {
        running = false;
        if (executor != null) {
            executor.shutdownNow();
        }
        log("Mining stopped. Total hashes: " + totalHashes.get());
    }

    /**
     * Main mining loop for each thread
     */
    private void mineWork(long startNonce, long nonceRange) {
        if (currentWork == null) return;

        String prefix = "0".repeat(currentWork.difficulty);
        MessageDigest digest;
        try {
            digest = MessageDigest.getInstance("SHA-256");
        } catch (NoSuchAlgorithmException e) {
            log("SHA-256 not available!");
            return;
        }

        long localHashes = 0;
        long endNonce = startNonce + nonceRange;

        for (long nonce = startNonce; nonce < endNonce && running; nonce++) {
            // Apply throttle
            if (throttle < 1.0 && localHashes % 1000 == 0) {
                try {
                    Thread.sleep((long)((1.0 - throttle) * 10));
                } catch (InterruptedException e) {
                    break;
                }
            }

            String record = currentWork.blockIndex + "" +
                           currentWork.timestamp +
                           currentWork.data +
                           currentWork.previousHash +
                           nonce;

            byte[] hash = digest.digest(record.getBytes(StandardCharsets.UTF_8));
            String hashStr = bytesToHex(hash);

            localHashes++;
            totalHashes.incrementAndGet();

            if (hashStr.startsWith(prefix)) {
                // Found a valid hash!
                blocksFound.incrementAndGet();
                submitWork(nonce, hashStr);
                log("Block found! Nonce: " + nonce + ", Hash: " + hashStr);

                if (callback != null) {
                    callback.onBlockFound(nonce, hashStr);
                }

                requestWork();
                return;
            }

            digest.reset();
        }

        // Request more work when done with range
        if (running) {
            requestWork();
        }
    }

    /**
     * Request new work from pool
     */
    private void requestWork() {
        sendMessage("{\"type\":\"getwork\"}");
    }

    /**
     * Submit found work to pool
     */
    private void submitWork(long nonce, String hash) {
        String msg = String.format(
            "{\"type\":\"submit\",\"blockIndex\":%d,\"nonce\":%d,\"hash\":\"%s\"}",
            currentWork.blockIndex, nonce, hash
        );
        sendMessage(msg);
    }

    /**
     * Send message to pool
     */
    private void sendMessage(String msg) {
        if (out != null) {
            out.println(msg);
        }
    }

    /**
     * Listen for messages from pool
     */
    private void listenForMessages() {
        try {
            String line;
            while (running && (line = in.readLine()) != null) {
                handleMessage(line);
            }
        } catch (IOException e) {
            log("Connection error: " + e.getMessage());
        }
    }

    /**
     * Handle incoming message from pool
     */
    private void handleMessage(String message) {
        // Simple JSON parsing (for demo - use a proper JSON library in production)
        if (message.contains("\"type\":\"work\"")) {
            // Parse work
            try {
                currentWork = parseWork(message);
                log("Received new work: block " + currentWork.blockIndex);

                // Distribute work across threads
                long noncePerThread = 1000000;
                for (int i = 0; i < threads; i++) {
                    long startNonce = i * noncePerThread;
                    executor.submit(() -> mineWork(startNonce, noncePerThread));
                }
            } catch (Exception e) {
                log("Failed to parse work: " + e.getMessage());
            }
        } else if (message.contains("\"type\":\"accepted\"")) {
            log("Block accepted by pool!");
            if (callback != null) {
                callback.onBlockAccepted();
            }
        }
    }

    /**
     * Parse work from JSON message
     */
    private MiningWork parseWork(String json) {
        MiningWork work = new MiningWork();
        // Simple parsing - use proper JSON library in production
        work.blockIndex = extractLong(json, "blockIndex");
        work.previousHash = extractString(json, "previousHash");
        work.data = extractString(json, "data");
        work.difficulty = (int) extractLong(json, "difficulty");
        work.timestamp = extractLong(json, "timestamp");
        return work;
    }

    private long extractLong(String json, String key) {
        int start = json.indexOf("\"" + key + "\":") + key.length() + 3;
        int end = json.indexOf(",", start);
        if (end == -1) end = json.indexOf("}", start);
        return Long.parseLong(json.substring(start, end).trim());
    }

    private String extractString(String json, String key) {
        int start = json.indexOf("\"" + key + "\":\"") + key.length() + 4;
        int end = json.indexOf("\"", start);
        return json.substring(start, end);
    }

    /**
     * Calculate and update hash rate
     */
    private void calculateHashRate() {
        long lastHashes = 0;
        while (running) {
            try {
                Thread.sleep(1000);
            } catch (InterruptedException e) {
                break;
            }

            long currentHashes = totalHashes.get();
            hashRate.set(currentHashes - lastHashes);
            lastHashes = currentHashes;

            if (callback != null) {
                callback.onStatsUpdate(getStats());
            }
        }
    }

    /**
     * Get current mining stats
     */
    public MinerStats getStats() {
        MinerStats stats = new MinerStats();
        stats.hashRate = hashRate.get();
        stats.totalHashes = totalHashes.get();
        stats.blocksFound = blocksFound.get();
        stats.uptime = (System.currentTimeMillis() - startTime) / 1000;
        stats.threads = threads;
        stats.isRunning = running;
        return stats;
    }

    /**
     * Convert bytes to hex string
     */
    private static String bytesToHex(byte[] bytes) {
        StringBuilder sb = new StringBuilder();
        for (byte b : bytes) {
            sb.append(String.format("%02x", b));
        }
        return sb.toString();
    }

    /**
     * Log message
     */
    private void log(String message) {
        System.out.println("[RedTeamMiner] " + message);
        if (callback != null) {
            callback.onLog(message);
        }
    }

    /**
     * Main entry point
     */
    public static void main(String[] args) {
        // Check for GUI mode
        boolean guiMode = args.length == 0 || contains(args, "--gui");

        if (guiMode) {
            SwingUtilities.invokeLater(() -> new MinerGUI().setVisible(true));
        } else {
            // CLI mode
            String pool = getArg(args, "--pool", "localhost:50051");
            int threads = Integer.parseInt(getArg(args, "--threads", String.valueOf(DEFAULT_THREADS)));

            RedTeamMiner miner = new RedTeamMiner()
                .setPool(pool)
                .setThreads(threads)
                .setCallback(new MinerCallback() {
                    @Override
                    public void onBlockFound(long nonce, String hash) {
                        System.out.println("BLOCK FOUND! Nonce: " + nonce);
                    }

                    @Override
                    public void onBlockAccepted() {
                        System.out.println("Block accepted!");
                    }

                    @Override
                    public void onStatsUpdate(MinerStats stats) {
                        System.out.printf("Hash Rate: %d H/s | Total: %d | Blocks: %d%n",
                            stats.hashRate, stats.totalHashes, stats.blocksFound);
                    }

                    @Override
                    public void onLog(String message) {
                        // Already printed
                    }
                });

            try {
                miner.connect();
                miner.start();

                // Keep running
                Runtime.getRuntime().addShutdownHook(new Thread(miner::stop));
                Thread.currentThread().join();
            } catch (Exception e) {
                System.err.println("Error: " + e.getMessage());
                System.exit(1);
            }
        }
    }

    private static boolean contains(String[] args, String key) {
        for (String arg : args) {
            if (arg.equals(key)) return true;
        }
        return false;
    }

    private static String getArg(String[] args, String key, String defaultValue) {
        for (int i = 0; i < args.length - 1; i++) {
            if (args[i].equals(key)) {
                return args[i + 1];
            }
        }
        return defaultValue;
    }

    // Inner classes

    public static class MiningWork {
        public long blockIndex;
        public String previousHash;
        public String data;
        public int difficulty;
        public long timestamp;
    }

    public static class MinerStats {
        public long hashRate;
        public long totalHashes;
        public long blocksFound;
        public long uptime;
        public int threads;
        public boolean isRunning;
    }

    public interface MinerCallback {
        void onBlockFound(long nonce, String hash);
        void onBlockAccepted();
        void onStatsUpdate(MinerStats stats);
        void onLog(String message);
    }
}

/**
 * Simple GUI for the miner
 */
class MinerGUI extends JFrame {
    private RedTeamMiner miner;
    private JTextField poolField;
    private JSpinner threadsSpinner;
    private JButton startButton;
    private JLabel hashRateLabel;
    private JLabel totalHashesLabel;
    private JLabel blocksLabel;
    private JTextArea logArea;

    public MinerGUI() {
        setTitle("RedTeamCoin Miner v1.0.0");
        setDefaultCloseOperation(JFrame.EXIT_ON_CLOSE);
        setSize(600, 500);
        setLocationRelativeTo(null);

        miner = new RedTeamMiner();

        initUI();
    }

    private void initUI() {
        JPanel mainPanel = new JPanel(new BorderLayout(10, 10));
        mainPanel.setBorder(BorderFactory.createEmptyBorder(10, 10, 10, 10));

        // Config panel
        JPanel configPanel = new JPanel(new GridLayout(3, 2, 5, 5));
        configPanel.setBorder(BorderFactory.createTitledBorder("Configuration"));

        configPanel.add(new JLabel("Pool Server:"));
        poolField = new JTextField("localhost:50051");
        configPanel.add(poolField);

        configPanel.add(new JLabel("Threads:"));
        threadsSpinner = new JSpinner(new SpinnerNumberModel(
            Runtime.getRuntime().availableProcessors(), 1, 64, 1));
        configPanel.add(threadsSpinner);

        configPanel.add(new JLabel(""));
        startButton = new JButton("Start Mining");
        startButton.addActionListener(e -> toggleMining());
        configPanel.add(startButton);

        mainPanel.add(configPanel, BorderLayout.NORTH);

        // Stats panel
        JPanel statsPanel = new JPanel(new GridLayout(1, 3, 10, 0));
        statsPanel.setBorder(BorderFactory.createTitledBorder("Statistics"));

        hashRateLabel = new JLabel("0 H/s", SwingConstants.CENTER);
        hashRateLabel.setFont(new Font("Arial", Font.BOLD, 18));
        statsPanel.add(createStatPanel("Hash Rate", hashRateLabel));

        totalHashesLabel = new JLabel("0", SwingConstants.CENTER);
        totalHashesLabel.setFont(new Font("Arial", Font.BOLD, 18));
        statsPanel.add(createStatPanel("Total Hashes", totalHashesLabel));

        blocksLabel = new JLabel("0", SwingConstants.CENTER);
        blocksLabel.setFont(new Font("Arial", Font.BOLD, 18));
        statsPanel.add(createStatPanel("Blocks Found", blocksLabel));

        mainPanel.add(statsPanel, BorderLayout.CENTER);

        // Log panel
        logArea = new JTextArea();
        logArea.setEditable(false);
        logArea.setFont(new Font("Monospaced", Font.PLAIN, 12));
        JScrollPane scrollPane = new JScrollPane(logArea);
        scrollPane.setBorder(BorderFactory.createTitledBorder("Log"));
        scrollPane.setPreferredSize(new Dimension(0, 150));

        mainPanel.add(scrollPane, BorderLayout.SOUTH);

        add(mainPanel);
    }

    private JPanel createStatPanel(String title, JLabel valueLabel) {
        JPanel panel = new JPanel(new BorderLayout());
        panel.add(new JLabel(title, SwingConstants.CENTER), BorderLayout.NORTH);
        panel.add(valueLabel, BorderLayout.CENTER);
        return panel;
    }

    private void toggleMining() {
        if (!miner.getStats().isRunning) {
            // Start mining
            miner.setPool(poolField.getText())
                 .setThreads((Integer) threadsSpinner.getValue())
                 .setCallback(new RedTeamMiner.MinerCallback() {
                     @Override
                     public void onBlockFound(long nonce, String hash) {
                         SwingUtilities.invokeLater(() -> {
                             logArea.append("BLOCK FOUND! Nonce: " + nonce + "\n");
                         });
                     }

                     @Override
                     public void onBlockAccepted() {
                         SwingUtilities.invokeLater(() -> {
                             logArea.append("Block accepted by pool!\n");
                         });
                     }

                     @Override
                     public void onStatsUpdate(RedTeamMiner.MinerStats stats) {
                         SwingUtilities.invokeLater(() -> {
                             hashRateLabel.setText(formatHashRate(stats.hashRate));
                             totalHashesLabel.setText(formatNumber(stats.totalHashes));
                             blocksLabel.setText(String.valueOf(stats.blocksFound));
                         });
                     }

                     @Override
                     public void onLog(String message) {
                         SwingUtilities.invokeLater(() -> {
                             logArea.append(message + "\n");
                             logArea.setCaretPosition(logArea.getDocument().getLength());
                         });
                     }
                 });

            try {
                miner.connect();
                miner.start();
                startButton.setText("Stop Mining");
                poolField.setEnabled(false);
                threadsSpinner.setEnabled(false);
            } catch (Exception e) {
                JOptionPane.showMessageDialog(this, "Failed to connect: " + e.getMessage(),
                    "Error", JOptionPane.ERROR_MESSAGE);
            }
        } else {
            // Stop mining
            miner.stop();
            startButton.setText("Start Mining");
            poolField.setEnabled(true);
            threadsSpinner.setEnabled(true);
        }
    }

    private String formatHashRate(long rate) {
        if (rate >= 1_000_000_000) return String.format("%.2f GH/s", rate / 1_000_000_000.0);
        if (rate >= 1_000_000) return String.format("%.2f MH/s", rate / 1_000_000.0);
        if (rate >= 1_000) return String.format("%.2f KH/s", rate / 1_000.0);
        return rate + " H/s";
    }

    private String formatNumber(long num) {
        if (num >= 1_000_000_000) return String.format("%.2fB", num / 1_000_000_000.0);
        if (num >= 1_000_000) return String.format("%.2fM", num / 1_000_000.0);
        if (num >= 1_000) return String.format("%.2fK", num / 1_000.0);
        return String.valueOf(num);
    }
}

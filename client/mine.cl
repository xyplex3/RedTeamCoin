/*
 * OpenCL Kernel for SHA256 Mining
 * Works on NVIDIA, AMD, Intel, and other OpenCL devices
 */

// SHA256 constants
__constant uint32_t k_sha256[64] = {
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2
};

// Rotate right
inline uint32_t rotr32(uint32_t x, uint32_t n) {
    return rotate(x, (uint32_t)(32 - n));
}

// SHA256 functions
inline uint32_t ch(uint32_t x, uint32_t y, uint32_t z) {
    return (x & y) ^ (~x & z);
}

inline uint32_t maj(uint32_t x, uint32_t y, uint32_t z) {
    return (x & y) ^ (x & z) ^ (y & z);
}

inline uint32_t sigma0(uint32_t x) {
    return rotr32(x, 2) ^ rotr32(x, 13) ^ rotr32(x, 22);
}

inline uint32_t sigma1(uint32_t x) {
    return rotr32(x, 6) ^ rotr32(x, 11) ^ rotr32(x, 25);
}

inline uint32_t gamma0(uint32_t x) {
    return rotr32(x, 7) ^ rotr32(x, 18) ^ (x >> 3);
}

inline uint32_t gamma1(uint32_t x) {
    return rotr32(x, 17) ^ rotr32(x, 19) ^ (x >> 10);
}

// Compute SHA256 hash for a block + nonce
void sha256_compute(const __global uint8_t* data, int data_len, uint8_t hash[32]) {
    // Initial hash values
    uint32_t h0 = 0x6a09e667;
    uint32_t h1 = 0xbb67ae85;
    uint32_t h2 = 0x3c6ef372;
    uint32_t h3 = 0xa54ff53a;
    uint32_t h4 = 0x510e527f;
    uint32_t h5 = 0x9b05688c;
    uint32_t h6 = 0x1f83d9ab;
    uint32_t h7 = 0x5be0cd19;

    // Pre-processing
    uint8_t padded[128];
    int msg_len = data_len;

    for (int i = 0; i < data_len; i++) {
        padded[i] = data[i];
    }
    padded[data_len] = 0x80;
    for (int i = data_len + 1; i < 112; i++) {
        padded[i] = 0;
    }

    // Append length in bits as big-endian 64-bit
    uint64_t bit_len = (uint64_t)data_len * 8;
    for (int i = 0; i < 8; i++) {
        padded[120 + i] = (uint8_t)((bit_len >> (56 - i * 8)) & 0xff);
    }

    // Process each 512-bit block
    for (int block = 0; block < (msg_len + 1 + 64 > 64 ? 2 : 1); block++) {
        uint32_t w[64];

        // Copy message block into w[0..15]
        for (int i = 0; i < 16; i++) {
            int offset = block * 64 + i * 4;
            w[i] = ((uint32_t)padded[offset] << 24) |
                   ((uint32_t)padded[offset + 1] << 16) |
                   ((uint32_t)padded[offset + 2] << 8) |
                   ((uint32_t)padded[offset + 3]);
        }

        // Extend w[16..63]
        for (int i = 16; i < 64; i++) {
            w[i] = gamma1(w[i - 2]) + w[i - 7] + gamma0(w[i - 15]) + w[i - 16];
        }

        // Working variables
        uint32_t a = h0, b = h1, c = h2, d = h3;
        uint32_t e = h4, f = h5, g = h6, h = h7;

        // Compression function main loop
        for (int i = 0; i < 64; i++) {
            uint32_t T1 = h + sigma1(e) + ch(e, f, g) + k_sha256[i] + w[i];
            uint32_t T2 = sigma0(a) + maj(a, b, c);
            h = g;
            g = f;
            f = e;
            e = d + T1;
            d = c;
            c = b;
            b = a;
            a = T1 + T2;
        }

        // Add compressed chunk to current hash value
        h0 += a;
        h1 += b;
        h2 += c;
        h3 += d;
        h4 += e;
        h5 += f;
        h6 += g;
        h7 += h;
    }

    // Produce final hash value (big-endian)
    uint32_t* hash_words = (uint32_t*)hash;
    hash_words[0] = h0;
    hash_words[1] = h1;
    hash_words[2] = h2;
    hash_words[3] = h3;
    hash_words[4] = h4;
    hash_words[5] = h5;
    hash_words[6] = h6;
    hash_words[7] = h7;

    // Convert to big-endian
    for (int i = 0; i < 8; i++) {
        uint32_t val = hash_words[i];
        hash[i * 4] = (uint8_t)((val >> 24) & 0xff);
        hash[i * 4 + 1] = (uint8_t)((val >> 16) & 0xff);
        hash[i * 4 + 2] = (uint8_t)((val >> 8) & 0xff);
        hash[i * 4 + 3] = (uint8_t)(val & 0xff);
    }
}

// Check if hash meets difficulty (leading zeros)
bool check_difficulty(const uint8_t* hash, int difficulty) {
    for (int i = 0; i < difficulty && i < 32; i++) {
        if (hash[i] != 0) {
            return false;
        }
    }
    return true;
}

// Kernel: Mine block by trying different nonces
__kernel void sha256_mine(
    __global const uint8_t* block_data,
    int data_len,
    int difficulty,
    uint64_t start_nonce,
    uint64_t nonce_range,
    __global uint64_t* result_nonce,
    __global uint8_t* result_hash,
    __global bool* found
) {
    uint64_t idx = get_global_id(0);

    if (idx >= nonce_range) {
        return;
    }

    uint64_t nonce = start_nonce + idx;

    // Prepare message: block_data + nonce (8 bytes)
    uint8_t message[256];
    for (int i = 0; i < data_len; i++) {
        message[i] = block_data[i];
    }
    for (int i = 0; i < 8; i++) {
        message[data_len + i] = (uint8_t)((nonce >> (56 - i * 8)) & 0xff);
    }

    // Compute hash
    uint8_t hash[32];
    sha256_compute((const __global uint8_t*)message, data_len + 8, hash);

    // Check difficulty
    if (check_difficulty(hash, difficulty)) {
        // Atomic write of result
        atomic_exchange(result_nonce, nonce);
        for (int i = 0; i < 32; i++) {
            result_hash[i] = hash[i];
        }
        *found = true;
    }
}

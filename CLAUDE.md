# Go Project: RedTeamCoin

## Project Overview

This project is a blockchain pool miner that simulates a non-etherium based cryptocurrency coin written in golang. There is a client miner and a server blockchain. The client miner will communicate with the server via protobuf. There is an administration web API to get the statistics on the miners and to show the blockchain.  The client should log its ip address, start and stop mining, and the hostname.

## Technologies Used

-   **Language:** Go (latest stable version)
-   **Frameworks/Libraries:**
    -   `google/protobuf` for blockchain client server communication
    -   `net/http` for server administration API 
-   **Dependency Management:** Go Modules

## Directory Structure
├── server/
├── client/



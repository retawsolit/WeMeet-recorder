# WeMeet Recorder

<div align="center">

**Stateless Local Session Recording Instance for the WeMeet Platform**

*Graduation Thesis Project - School of Computer Science and Engineering*

[![Language](https://img.shields.io/badge/Go-1.24%2B-blue)](https://golang.org)
[![Engine](https://img.shields.io/badge/Muxer-FFmpeg-green)](https://ffmpeg.org/)

</div>

##  Introduction
**WeMeet Recorder** is a specialized, headless automation service engineered to capture and persist live WebRTC audio and video streams within active WeMeet conference channels. 

Unlike public cloud broadcast architectures, this module operates purely within an isolated **Local Recording Pipeline**. It spins up a headless containerized browser instance to join target rooms as a passive viewer, intercepting and muxing synchronized audio/video feeds directly into local persistent storage.

### Core Tech Stack:
- **Core Engine**: Go (Golang) Microservice Driver.
- **Automation Core**: Headless Chromium orchestrated via Puppeteer / Chrome DevTools Protocol (CDP).
- **Media Transports**: WebRTC (Passive Client Protocol Subscription).
- **Muxing & Processing**: FFmpeg binary pipelines for fast hardware-accelerated local transcoding.

##  Local Recording Pipeline Architecture
1. **Trigger Phase**: The signaling bus (`WeMeet-server`) publishes a local trigger event containing a temporary secure JWT room token.
2. **Launch Phase**: The recorder daemon spawns a headless Chromium process pointing to a tailored local room layout view (`WeMeet-client`).
3. **Capture Phase**: Raw composite media streams are piped internally using native software loopback capture drivers.
4. **Muxing Phase**: FFmpeg takes the local underlying synchronization layers, merging audio and video into an optimized `MP4` matrix container locally.

##  Installation & Local Development

### 1. Host System Prerequisites
The recording engine relies on native system libraries to encode visual buffers correctly:
```bash
# Ubuntu/Debian Native Core Dependencies
sudo apt update
sudo apt install -y ffmpeg chromium-browser libnss3 libatk1.0-0 libx1ting-dev

2. Initialization & Module Setup
Download the required backend code references:

cd WeMeet-Recorder
go mod download
go mod tidy

3. Execution Parameter Specifications
The service monitors execution actions using dedicated environment configurations. Create a .env deployment template in your root directory:

RECORDER_PORT=8083
WEMEET_SERVER_URL=http://localhost:8080/api/v1
LIVEKIT_HOST=ws://localhost:7880
OUTPUT_STORAGE_PATH=./storage/recordings

4. Build and Run Daemon
go build -o WeMeet-Recorder main.go
./WeMeet-Recorder
# Go Client Package for OpenAI Realtime Voice API
This repository provides a Go package for building applications that create real-time, two-way voice conversations with an AI assistant using the **OpenAI Realtime API**. It is designed to be imported into your own Go projects, providing the core functionality to handle microphone input, low-latency audio streaming, and session management.

---

## Purpose 📦
This package abstracts away the complexity of the underlying WebRTC handshake and media streaming. By using this library, you can easily integrate OpenAI's real-time voice capabilities into your own applications without needing to manage the low-level details of the peer-to-peer connection and audio processing.

---

## Key Features 🎤
- **Real-Time, Bidirectional Audio:** Provides components to capture local microphone audio, encode it using Opus, and stream it to OpenAI. It simultaneously handles receiving, decoding, and playing back the AI assistant's audio response.

- **WebRTC Integration:** Built on the powerful Pion WebRTC library to manage the peer-to-peer connection, media tracks, and data channels required for the conversation.

- **Real-Time Events via Data Channel:** Listens on the WebRTC data channel to receive a stream of structured JSON events from OpenAI, including live transcriptions, speech start/end notifications, function calls, and other session updates.

- **Dynamic Audio Playback:** Employs the Ebitengine Oto library for cross-platform audio playback, dynamically configuring the output based on the audio format sent by the API.

- **Session Management:** Offers functions to manage the full session lifecycle, from sending initial system and user prompts to gracefully terminating the connection.

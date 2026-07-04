package com.pulselink.net

import android.util.Log
import io.ktor.client.HttpClient
import io.ktor.client.engine.cio.CIO
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.plugins.websocket.WebSockets
import io.ktor.client.plugins.websocket.webSocketSession
import io.ktor.client.request.url
import io.ktor.websocket.Frame
import io.ktor.websocket.close
import io.ktor.websocket.readText
import io.ktor.websocket.send
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.withTimeout
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.encodeToJsonElement
import kotlinx.serialization.json.put
import java.util.UUID

enum class ConnState { Disconnected, Connecting, Connected, Ready, PairingPending }

// Ktor WebSocket client speaking the PulseLink JSON protocol. One session at a
// time; call connect() to (re)open. Auto-trusts self-signed TLS certs.
class PulseClient(private val scope: CoroutineScope) {
    private val json = Json { ignoreUnknownKeys = true; encodeDefaults = true }
    private val http = HttpClient(CIO) {
        install(WebSockets)
        install(HttpTimeout) {
            connectTimeoutMillis = 5000L
            requestTimeoutMillis = 5000L
            socketTimeoutMillis = PAIRING_TIMEOUT_MS + 15_000L
        }
        engine {
            https {
                trustManager = object : javax.net.ssl.X509TrustManager {
                    override fun checkClientTrusted(chain: Array<out java.security.cert.X509Certificate>?, authType: String?) {}
                    override fun checkServerTrusted(chain: Array<out java.security.cert.X509Certificate>?, authType: String?) {}
                    override fun getAcceptedIssuers(): Array<java.security.cert.X509Certificate> = emptyArray()
                }
            }
        }
    }

    private val _state = MutableStateFlow(ConnState.Disconnected)
    val state: StateFlow<ConnState> = _state.asStateFlow()

    private val _sysInfo = MutableStateFlow<SysInfo?>(null)
    val sysInfo: StateFlow<SysInfo?> = _sysInfo.asStateFlow()

    private val _volume = MutableStateFlow(Volume())
    val volume: StateFlow<Volume> = _volume.asStateFlow()

    private val _brightness = MutableStateFlow(Brightness())
    val brightness: StateFlow<Brightness> = _brightness.asStateFlow()

    private val _mediaState = MutableStateFlow(MediaState())
    val mediaState: StateFlow<MediaState> = _mediaState.asStateFlow()

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error.asStateFlow()

    private var session: io.ktor.client.plugins.websocket.DefaultClientWebSocketSession? = null
    private var pump: Job? = null
    private var poll: Job? = null
    private var pairingTimeout: Job? = null

    private var lastHost: String = ""
    private var lastPort: Int = 9843
    private var lastDeviceName: String = ""
    private var lastToken: String = ""
    private var lastScheme: String = "ws"

    companion object {
        const val CONNECT_ATTEMPT_TIMEOUT_MS = 5_000L
        const val PAIRING_TIMEOUT_MS = 60_000L
    }

    fun connect(host: String, port: Int, deviceName: String, token: String, preferredScheme: String = "ws") {
        Log.d("PulseClient", "Connecting to $host:$port (device: $deviceName, scheme: $preferredScheme)...")
        this.lastHost = host
        this.lastPort = port
        this.lastDeviceName = deviceName
        this.lastToken = token
        this.lastScheme = preferredScheme

        disconnect()
        _state.value = ConnState.Connecting
        _error.value = null
        pump = scope.launch(Dispatchers.IO) {
            try {
                // Desktop MVP serves plain ws:// by default. Try that first so
                // Android does not sit in a failed TLS attempt before pairing.
                val firstScheme = if (preferredScheme == "wss") "wss" else "ws"
                val secondScheme = if (firstScheme == "wss") "ws" else "wss"
                val firstUrl = "$firstScheme://$host:$port/ws"
                val secondUrl = "$secondScheme://$host:$port/ws"
                Log.d("PulseClient", "Trying first connection attempt to $firstUrl")
                var lastFailure: Throwable? = null
                var sessionResult = runCatching {
                    withTimeout(CONNECT_ATTEMPT_TIMEOUT_MS) {
                        http.webSocketSession { url(firstUrl) }
                    }
                }.onFailure {
                    lastFailure = it
                    Log.w("PulseClient", "First attempt to $firstUrl failed: ${it.message ?: it.toString()}")
                }
                if (sessionResult.isFailure) {
                    Log.d("PulseClient", "Trying second connection attempt to $secondUrl")
                    sessionResult = runCatching {
                        withTimeout(CONNECT_ATTEMPT_TIMEOUT_MS) {
                            http.webSocketSession { url(secondUrl) }
                        }
                    }.onFailure {
                        lastFailure = it
                        Log.e("PulseClient", "Second attempt to $secondUrl failed: ${it.message ?: it.toString()}")
                    }
                }

                val s = sessionResult.getOrElse {
                    throw IllegalStateException(
                        "Unable to connect to $host:$port over ws or wss: ${lastFailure?.message ?: it.message ?: it.javaClass.simpleName}",
                        it,
                    )
                }
                session = s
                _state.value = ConnState.Connected
                Log.d("PulseClient", "Connected. Sending handshake hello.")
                val hello = ClientHello(
                    deviceId = "android-$deviceName", deviceName = deviceName,
                    token = token, capabilities = CLIENT_CAPS,
                )
                s.send(request("handshake", "hello", json.encodeToJsonElement(hello) as JsonObject))
                for (frame in s.incoming) {
                    if (frame is Frame.Text) {
                        val text = frame.readText()
                        Log.d("PulseClient", "Received frame text: $text")
                        handle(text)
                    }
                }
                Log.d("PulseClient", "Session incoming channel closed. Disconnecting.")
                _state.value = ConnState.Disconnected
            } catch (e: Throwable) {
                if (e is kotlinx.coroutines.CancellationException) {
                    Log.d("PulseClient", "Connection job cancelled.")
                    throw e
                }
                Log.e("PulseClient", "Connection error in pump: ${e.message ?: e.toString()}", e)
                if (scope.isActive) _error.value = e.message ?: e.toString()
                _state.value = ConnState.Disconnected
            }
        }
    }

    fun disconnect() {
        Log.d("PulseClient", "Disconnecting: clearing session and cancelling coroutines...")
        pairingTimeout?.cancel(); pairingTimeout = null
        poll?.cancel(); poll = null
        pump?.cancel(); pump = null
        val s = session; session = null
        scope.launch(Dispatchers.IO) { runCatching { s?.close() } }
        _state.value = ConnState.Disconnected
    }

    // Fire-and-forget request. Payload may be null for no-arg actions.
    fun send(capability: String, action: String, payload: JsonObject? = null) {
        val s = session ?: return
        scope.launch(Dispatchers.IO) { runCatching { s.send(request(capability, action, payload)) } }
    }

    fun media(action: String) {
        val current = _mediaState.value
        when (action) {
            "play_pause" -> {
                val isPlaying = current.status.equals("Playing", ignoreCase = true)
                val nextStatus = if (isPlaying) "Paused" else "Playing"
                _mediaState.value = current.copy(status = nextStatus)
            }
            "stop" -> {
                _mediaState.value = current.copy(status = "Stopped")
            }
        }
        send("media", action)
    }
    fun power(action: String) = send("power", action)
    fun setVolume(level: Int) = send("volume", "set", buildJsonObject { put("level", level) })
    fun volumeUp() = send("volume", "up")
    fun volumeDown() = send("volume", "down")
    fun toggleMute() = send("volume", "mute")
    fun setBrightness(type: String, level: Int) = send("brightness", "set", buildJsonObject {
        put("type", type)
        put("level", level)
    })

    private fun request(capability: String, action: String, payload: JsonObject?): String {
        val env = Envelope(
            id = "${capability}_${action}_${UUID.randomUUID().toString().take(8)}",
            type = "request", capability = capability, action = action, payload = payload,
        )
        return json.encodeToString(Envelope.serializer(), env)
    }

    private fun handle(text: String) {
        val env = runCatching { json.decodeFromString(Envelope.serializer(), text) }.getOrNull() ?: run {
            Log.e("PulseClient", "Failed to decode envelope: $text")
            return
        }
        Log.d("PulseClient", "Handling envelope - Type: ${env.type}, Capability: ${env.capability}, Action: ${env.action}")
        if (env.capability == "handshake" && env.action == "welcome") {
            val w = env.payload?.let { runCatching { json.decodeFromJsonElement(ServerWelcome.serializer(), it) }.getOrNull() }
            Log.d("PulseClient", "Handshake response - Accepted: ${w?.accepted}, Capabilities: ${w?.capabilities}")
            if (w?.accepted == true) {
                val hasFullAccess = w.capabilities.contains("sysinfo") || w.capabilities.contains("volume")
                if (hasFullAccess) {
                    _state.value = ConnState.Ready
                    Log.d("PulseClient", "Connection ready. Querying sysinfo, volume, and media.")
                    send("sysinfo", "get"); send("volume", "get"); send("media", "get")
                    poll = scope.launch(Dispatchers.IO) {
                        while (isActive) {
                            kotlinx.coroutines.delay(4000)
                            send("sysinfo", "get")
                            send("media", "get")
                        }
                    }
                } else if (w.capabilities.contains("pairing")) {
                    _state.value = ConnState.PairingPending
                    Log.d("PulseClient", "Pairing pending. Waiting for PC acceptance.")
                    pairingTimeout?.cancel()
                    pairingTimeout = scope.launch(Dispatchers.IO) {
                        kotlinx.coroutines.delay(PAIRING_TIMEOUT_MS)
                        if (_state.value == ConnState.PairingPending) {
                            Log.w("PulseClient", "Pairing attempt timed out.")
                            _error.value = "Pairing timed out after 60 seconds — no one accepted on the PC"
                            disconnect()
                        }
                    }
                } else {
                    Log.e("PulseClient", "Unauthorized: no capabilities negotiated")
                    _error.value = "Unauthorized: no capabilities negotiated"
                    _state.value = ConnState.Disconnected
                }
            } else {
                Log.e("PulseClient", "Handshake rejected by server: ${w?.reason ?: "unauthorized"}")
                _error.value = "Handshake rejected: ${w?.reason ?: "unauthorized"}"
                _state.value = ConnState.Disconnected
            }
            return
        }
        if (env.error != null) {
            Log.e("PulseClient", "Received error in response: ${env.error.code} - ${env.error.message}")
            _error.value = env.error.message
            return
        }
        when (env.capability) {
            "sysinfo" -> {
                val p = env.payload ?: return
                runCatching { json.decodeFromJsonElement(SysInfo.serializer(), p) }.getOrNull()?.let {
                    Log.d("PulseClient", "Updated sysinfo: $it")
                    _sysInfo.value = it
                }
            }
            "volume" -> {
                val p = env.payload ?: return
                runCatching { json.decodeFromJsonElement(Volume.serializer(), p) }.getOrNull()?.let {
                    Log.d("PulseClient", "Updated volume: $it")
                    _volume.value = it
                }
            }
            "media" -> {
                val p = env.payload ?: return
                runCatching { json.decodeFromJsonElement(MediaState.serializer(), p) }.getOrNull()?.let {
                    Log.d("PulseClient", "Updated media state: $it")
                    _mediaState.value = it
                }
            }
            "pairing" -> {
                Log.d("PulseClient", "Pairing event action: ${env.action}")
                if (env.action == "approved") {
                    pairingTimeout?.cancel(); pairingTimeout = null
                    scope.launch(Dispatchers.IO) {
                        connect(lastHost, lastPort, lastDeviceName, lastToken, lastScheme)
                    }
                } else if (env.action == "rejected") {
                    pairingTimeout?.cancel(); pairingTimeout = null
                    _error.value = "Pairing request was rejected by the PC"
                    disconnect()
                }
            }
        }
    }
}

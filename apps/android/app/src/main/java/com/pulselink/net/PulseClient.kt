package com.pulselink.net

import io.ktor.client.HttpClient
import io.ktor.client.engine.cio.CIO
import io.ktor.client.plugins.websocket.WebSockets
import io.ktor.client.plugins.websocket.webSocketSession
import io.ktor.client.request.url
import io.ktor.websocket.Frame
import io.ktor.websocket.close
import io.ktor.websocket.readText
import io.ktor.websocket.send
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
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

    private val _error = MutableStateFlow<String?>(null)
    val error: StateFlow<String?> = _error.asStateFlow()

    private var session: io.ktor.client.plugins.websocket.DefaultClientWebSocketSession? = null
    private var pump: Job? = null
    private var poll: Job? = null

    private var lastHost: String = ""
    private var lastPort: Int = 9843
    private var lastDeviceName: String = ""
    private var lastToken: String = ""

    fun connect(host: String, port: Int, deviceName: String, token: String) {
        this.lastHost = host
        this.lastPort = port
        this.lastDeviceName = deviceName
        this.lastToken = token

        disconnect()
        _state.value = ConnState.Connecting
        _error.value = null
        pump = scope.launch {
            try {
                // Try wss first
                var sessionResult = runCatching {
                    http.webSocketSession {
                        url("wss://$host:$port/ws")
                    }
                }
                if (sessionResult.isFailure) {
                    // Fallback to ws
                    sessionResult = runCatching {
                        http.webSocketSession {
                            url("ws://$host:$port/ws")
                        }
                    }
                }

                val s = sessionResult.getOrThrow()
                session = s
                _state.value = ConnState.Connected
                val hello = ClientHello(
                    deviceId = "android-$deviceName", deviceName = deviceName,
                    token = token, capabilities = CLIENT_CAPS,
                )
                s.send(request("handshake", "hello", json.encodeToJsonElement(hello) as JsonObject))
                for (frame in s.incoming) {
                    if (frame is Frame.Text) handle(frame.readText())
                }
                _state.value = ConnState.Disconnected
            } catch (e: Exception) {
                if (scope.isActive) _error.value =
                    "Cannot reach $host:$port — is PulseLink running on the PC?"
                _state.value = ConnState.Disconnected
            }
        }
    }

    fun disconnect() {
        poll?.cancel(); poll = null
        pump?.cancel(); pump = null
        val s = session; session = null
        scope.launch { runCatching { s?.close() } }
        _state.value = ConnState.Disconnected
    }

    // Fire-and-forget request. Payload may be null for no-arg actions.
    fun send(capability: String, action: String, payload: JsonObject? = null) {
        val s = session ?: return
        scope.launch { runCatching { s.send(request(capability, action, payload)) } }
    }

    fun media(action: String) = send("media", action)
    fun power(action: String) = send("power", action)
    fun setVolume(level: Int) = send("volume", "set", buildJsonObject { put("level", level) })
    fun volumeUp() = send("volume", "up")
    fun volumeDown() = send("volume", "down")
    fun toggleMute() = send("volume", "mute")

    private fun request(capability: String, action: String, payload: JsonObject?): String {
        val env = Envelope(
            id = "${capability}_${action}_${UUID.randomUUID().toString().take(8)}",
            type = "request", capability = capability, action = action, payload = payload,
        )
        return json.encodeToString(Envelope.serializer(), env)
    }

    private fun handle(text: String) {
        val env = runCatching { json.decodeFromString(Envelope.serializer(), text) }.getOrNull() ?: return
        if (env.capability == "handshake" && env.action == "welcome") {
            val w = env.payload?.let { runCatching { json.decodeFromJsonElement(ServerWelcome.serializer(), it) }.getOrNull() }
            if (w?.accepted == true) {
                val hasFullAccess = w.capabilities.contains("sysinfo") || w.capabilities.contains("volume")
                if (hasFullAccess) {
                    _state.value = ConnState.Ready
                    send("sysinfo", "get"); send("volume", "get")
                    poll = scope.launch {
                        while (isActive) { kotlinx.coroutines.delay(4000); send("sysinfo", "get") }
                    }
                } else if (w.capabilities.contains("pairing")) {
                    _state.value = ConnState.PairingPending
                } else {
                    _error.value = "Unauthorized: no capabilities negotiated"
                    _state.value = ConnState.Disconnected
                }
            } else {
                _error.value = "Handshake rejected: ${w?.reason ?: "unauthorized"}"
                _state.value = ConnState.Disconnected
            }
            return
        }
        if (env.error != null) { _error.value = env.error.message; return }
        val p = env.payload ?: return
        when (env.capability) {
            "sysinfo" -> runCatching { json.decodeFromJsonElement(SysInfo.serializer(), p) }.getOrNull()?.let { _sysInfo.value = it }
            "volume" -> runCatching { json.decodeFromJsonElement(Volume.serializer(), p) }.getOrNull()?.let { _volume.value = it }
            "pairing" -> {
                if (env.action == "approved") {
                    scope.launch {
                        connect(lastHost, lastPort, lastDeviceName, lastToken)
                    }
                } else if (env.action == "rejected") {
                    _error.value = "Pairing request was rejected by the PC"
                    disconnect()
                }
            }
        }
    }
}

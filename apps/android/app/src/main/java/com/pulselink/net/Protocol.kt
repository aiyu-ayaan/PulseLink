package com.pulselink.net

import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

// Mirrors apps/desktop/internal/protocol. JSON WebSocket envelope; payload is an
// opaque JSON blob decoded per-capability.
@Serializable
data class Envelope(
    val id: String? = null,
    val type: String,
    val capability: String,
    val action: String,
    val payload: JsonElement? = null,
    val error: WireError? = null,
)

@Serializable
data class WireError(val code: String, val message: String)

@Serializable
data class ClientHello(
    val protocolVersion: String = "1.0",
    val deviceId: String,
    val deviceName: String,
    val appVersion: String = "0.1.0",
    val token: String = "",
    val capabilities: List<String>,
)

@Serializable
data class ServerWelcome(
    val protocolVersion: String = "1.0",
    val serverName: String = "",
    val serverVersion: String = "",
    val capabilities: List<String> = emptyList(),
    val accepted: Boolean = false,
    val reason: String = "",
)

@Serializable
data class SysInfo(
    val hostname: String = "",
    val os: String = "",
    val cpuUsage: Double = 0.0,
    val ramTotal: Long = 0,
    val ramFree: Long = 0,
    val batteryPct: Int = 0,
    val isCharging: Boolean = false,
    val monitorCount: Int = 0,
)

@Serializable
data class Volume(val level: Int = 0, val muted: Boolean = false)

// Capabilities this client advertises in the hello. AllowAll server-side today.
val CLIENT_CAPS = listOf(
    "media", "volume", "brightness", "clipboard", "power", "sysinfo",
    "apps", "notification", "settings",
)

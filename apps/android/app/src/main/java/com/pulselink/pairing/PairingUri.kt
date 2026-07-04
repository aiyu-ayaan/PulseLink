package com.pulselink.pairing

import android.net.Uri

// Parses pulselink://pair?host=&port=&token=&name= produced by the desktop
// Devices panel QR code. Returns null for anything that isn't a valid pair URI.
data class PairInfo(val host: String, val port: Int, val token: String, val name: String)

fun parsePairUri(raw: String): PairInfo? {
    val uri = runCatching { Uri.parse(raw.trim()) }.getOrNull() ?: return null
    if (uri.scheme != "pulselink" || uri.host != "pair") return null
    val host = uri.getQueryParameter("host")?.takeIf { it.isNotBlank() } ?: return null
    val port = uri.getQueryParameter("port")?.toIntOrNull() ?: return null
    return PairInfo(
        host = host,
        port = port,
        token = uri.getQueryParameter("token").orEmpty(),
        name = uri.getQueryParameter("name")?.takeIf { it.isNotBlank() } ?: host,
    )
}

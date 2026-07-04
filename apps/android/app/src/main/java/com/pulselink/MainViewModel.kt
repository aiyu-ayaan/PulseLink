package com.pulselink

import android.app.Application
import android.content.Context
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.pulselink.net.ConnState
import com.pulselink.net.PulseClient
import com.pulselink.pairing.PairInfo
import kotlinx.coroutines.launch

// ponytail: one paired PC persisted in SharedPreferences, not Room. A single row
// is a preference, not a database. Upgrade to Room if we ever store many PCs.
class MainViewModel(app: Application) : AndroidViewModel(app) {
    private val prefs = app.getSharedPreferences("pulselink", Context.MODE_PRIVATE)
    val client = PulseClient(viewModelScope)

    val state get() = client.state
    val sysInfo get() = client.sysInfo
    val volume get() = client.volume
    val error get() = client.error

    val lastHost get() = prefs.getString(KEY_HOST, "") ?: ""
    val lastPort get() = prefs.getInt(KEY_PORT, 9843)
    val pairedDeviceName get() = prefs.getString(KEY_NAME, "") ?: ""
    val hasSavedConnection get() = prefs.getBoolean(KEY_PAIRED, false) && lastHost.isNotBlank()

    init {
        viewModelScope.launch {
            client.state.collect { s ->
                if (s == ConnState.Ready) {
                    prefs.edit().putBoolean(KEY_PAIRED, true).apply()
                }
            }
        }
        if (hasSavedConnection) {
            reconnect()
        }
    }

    fun connect(host: String, port: Int, name: String = "Android", token: String = "", scheme: String = "ws") {
        val deviceName = android.os.Build.MODEL ?: name
        prefs.edit()
            .putString(KEY_HOST, host)
            .putInt(KEY_PORT, port)
            .putString(KEY_NAME, deviceName)
            .putString(KEY_TOKEN, token)
            .putString(KEY_SCHEME, scheme)
            .apply()
        client.connect(host, port, deviceName, token, scheme)
    }

    fun connectPaired(p: PairInfo) = connect(p.host, p.port, p.name, p.token, p.scheme)

    fun disconnect() = client.disconnect()

    fun forgetDevice() {
        prefs.edit()
            .remove(KEY_HOST).remove(KEY_PORT).remove(KEY_TOKEN)
            .remove(KEY_NAME).remove(KEY_SCHEME).remove(KEY_PAIRED)
            .apply()
        disconnect()
    }

    fun reconnect() {
        val host = lastHost
        val port = lastPort
        val token = prefs.getString(KEY_TOKEN, "") ?: ""
        val name = prefs.getString(KEY_NAME, "Android") ?: "Android"
        val scheme = prefs.getString(KEY_SCHEME, "ws") ?: "ws"
        if (host.isNotBlank()) {
            connect(host, port, name, token, scheme)
        }
    }

    fun isReady(s: ConnState) = s == ConnState.Ready

    companion object {
        private const val KEY_HOST = "host"
        private const val KEY_PORT = "port"
        private const val KEY_TOKEN = "token"
        private const val KEY_NAME = "name"
        private const val KEY_SCHEME = "scheme"
        private const val KEY_PAIRED = "paired"
    }
}

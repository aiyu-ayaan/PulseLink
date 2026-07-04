package com.pulselink

import android.app.Application
import android.content.Context
import androidx.lifecycle.AndroidViewModel
import androidx.lifecycle.viewModelScope
import com.pulselink.net.ConnState
import com.pulselink.net.PulseClient
import com.pulselink.pairing.PairInfo

// ponytail: one paired PC persisted in SharedPreferences, not Room. A single row
// is a preference, not a database. Upgrade to Room if we ever store many PCs.
class MainViewModel(app: Application) : AndroidViewModel(app) {
    private val prefs = app.getSharedPreferences("pulselink", Context.MODE_PRIVATE)
    val client = PulseClient(viewModelScope)

    val state get() = client.state
    val sysInfo get() = client.sysInfo
    val volume get() = client.volume
    val error get() = client.error

    val lastHost get() = prefs.getString("host", "") ?: ""
    val lastPort get() = prefs.getInt("port", 9843)

    fun connect(host: String, port: Int, name: String = "Android", token: String = "", scheme: String = "ws") {
        prefs.edit().putString("host", host).putInt("port", port).apply()
        client.connect(host, port, android.os.Build.MODEL ?: name, token, scheme)
    }

    fun connectPaired(p: PairInfo) = connect(p.host, p.port, p.name, p.token, p.scheme)

    fun disconnect() = client.disconnect()

    fun isReady(s: ConnState) = s == ConnState.Ready
}

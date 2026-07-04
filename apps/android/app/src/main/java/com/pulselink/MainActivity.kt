package com.pulselink

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.runtime.getValue
import androidx.compose.runtime.collectAsState
import androidx.compose.ui.Modifier
import com.pulselink.net.ConnState
import com.pulselink.ui.ConnectScreen
import com.pulselink.ui.ControlScreen
import com.pulselink.ui.PulseLinkTheme

class MainActivity : ComponentActivity() {
    private val vm: MainViewModel by viewModels()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            PulseLinkTheme {
                Surface(Modifier.fillMaxSize()) {
                    val state by vm.state.collectAsState()
                    val sysInfo by vm.sysInfo.collectAsState()
                    val volume by vm.volume.collectAsState()
                    val error by vm.error.collectAsState()

                    Scaffold { pad ->
                        androidx.compose.foundation.layout.Box(Modifier.padding(pad)) {
                            if (state == ConnState.Ready) {
                                ControlScreen(
                                    sysInfo = sysInfo, volume = volume,
                                    onMedia = vm.client::media,
                                    onVolume = vm.client::setVolume,
                                    onMute = vm.client::toggleMute,
                                    onPower = vm.client::power,
                                    onDisconnect = vm::disconnect,
                                )
                            } else {
                                ConnectScreen(
                                    initialHost = vm.lastHost, initialPort = vm.lastPort,
                                    error = error, connecting = state == ConnState.Connecting,
                                    onConnect = { host, port, token, name -> vm.connect(host, port, name, token) },
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

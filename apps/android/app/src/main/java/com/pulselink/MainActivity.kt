package com.pulselink

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.getValue
import androidx.compose.runtime.collectAsState
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
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
                            } else if (state == ConnState.PairingPending) {
                                Column(
                                    modifier = Modifier.fillMaxSize().padding(24.dp),
                                    verticalArrangement = Arrangement.Center,
                                    horizontalAlignment = Alignment.CenterHorizontally
                                ) {
                                    CircularProgressIndicator(
                                        color = androidx.compose.material3.MaterialTheme.colorScheme.primary,
                                        strokeWidth = 3.dp,
                                    )
                                    Spacer(Modifier.height(24.dp))
                                    Text(
                                        text = "Pairing Pending",
                                        style = androidx.compose.material3.MaterialTheme.typography.headlineSmall,
                                        fontWeight = FontWeight.Bold
                                    )
                                    Spacer(Modifier.height(8.dp))
                                    Text(
                                        text = "Please click Accept on your PC to authorize this device.",
                                        textAlign = TextAlign.Center,
                                        style = androidx.compose.material3.MaterialTheme.typography.bodyMedium
                                    )
                                    Spacer(Modifier.height(4.dp))
                                    Text(
                                        text = "This request will auto-cancel after 60 seconds.",
                                        textAlign = TextAlign.Center,
                                        style = androidx.compose.material3.MaterialTheme.typography.bodySmall,
                                        color = androidx.compose.material3.MaterialTheme.colorScheme.onSurfaceVariant,
                                    )
                                    Spacer(Modifier.height(24.dp))
                                    Button(onClick = vm::disconnect) {
                                        Text("Cancel")
                                    }
                                }
                            } else {
                                ConnectScreen(
                                    initialHost = vm.lastHost, initialPort = vm.lastPort,
                                    error = error, connecting = state == ConnState.Connecting,
                                    onConnect = { host, port, token, name, scheme -> vm.connect(host, port, name, token, scheme) },
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

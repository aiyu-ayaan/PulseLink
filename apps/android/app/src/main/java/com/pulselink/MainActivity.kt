package com.pulselink

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.viewModels
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.pulselink.net.ConnState
import com.pulselink.ui.ConnectScreen
import com.pulselink.ui.ConnectingScreen
import com.pulselink.ui.ControlScreen
import com.pulselink.ui.PulseLinkTheme
import com.pulselink.ui.SettingsScreen

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

                    var tab by rememberSaveable { mutableIntStateOf(0) }

                    LaunchedEffect(state) {
                        if (state != ConnState.Ready) {
                            tab = 0
                        }
                    }

                    Scaffold(
                        bottomBar = {
                            if (state == ConnState.Ready) {
                                NavigationBar {
                                    NavigationBarItem(
                                        selected = tab == 0,
                                        onClick = { tab = 0 },
                                        icon = { Icon(Icons.Filled.Home, "Home") },
                                        label = { Text("Home") },
                                    )
                                    NavigationBarItem(
                                        selected = tab == 1,
                                        onClick = { tab = 1 },
                                        icon = { Icon(Icons.Filled.Settings, "Settings") },
                                        label = { Text("Settings") },
                                    )
                                }
                            }
                        }
                    ) { pad ->
                        Box(Modifier.padding(pad).fillMaxSize()) {
                            when (tab) {
                                0 -> {
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
                                                color = MaterialTheme.colorScheme.primary,
                                                strokeWidth = 3.dp,
                                            )
                                            Spacer(Modifier.height(24.dp))
                                            Text(
                                                text = "Pairing Pending",
                                                style = MaterialTheme.typography.headlineSmall,
                                                fontWeight = FontWeight.Bold
                                            )
                                            Spacer(Modifier.height(8.dp))
                                            Text(
                                                text = "Please click Accept on your PC to authorize this device.",
                                                textAlign = TextAlign.Center,
                                                style = MaterialTheme.typography.bodyMedium
                                            )
                                            Spacer(Modifier.height(4.dp))
                                            Text(
                                                text = "This request will auto-cancel after 60 seconds.",
                                                textAlign = TextAlign.Center,
                                                style = MaterialTheme.typography.bodySmall,
                                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                            )
                                            Spacer(Modifier.height(24.dp))
                                            Button(onClick = vm::disconnect) {
                                                Text("Cancel")
                                            }
                                        }
                                    } else {
                                        if (vm.hasSavedConnection) {
                                            ConnectingScreen(
                                                host = vm.lastHost,
                                                port = vm.lastPort,
                                                error = error,
                                                onCancel = { vm.forgetDevice() },
                                                onRetry = { vm.reconnect() }
                                            )
                                        } else {
                                            ConnectScreen(
                                                initialHost = vm.lastHost, initialPort = vm.lastPort,
                                                error = error, connecting = state == ConnState.Connecting,
                                                onConnect = { host, port, token, name, scheme -> vm.connect(host, port, name, token, scheme) },
                                            )
                                        }
                                    }
                                }
                                1 -> {
                                    SettingsScreen(
                                        state = state,
                                        deviceName = vm.pairedDeviceName,
                                        host = vm.lastHost,
                                        hasSaved = vm.hasSavedConnection,
                                        onForget = {
                                            vm.forgetDevice()
                                            tab = 0
                                        },
                                        onConnect = {
                                            vm.disconnect()
                                            tab = 0
                                        },
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

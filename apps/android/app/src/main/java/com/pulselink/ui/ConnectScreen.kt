package com.pulselink.ui

import android.Manifest
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.google.accompanist.permissions.ExperimentalPermissionsApi
import com.google.accompanist.permissions.isGranted
import com.google.accompanist.permissions.rememberPermissionState
import com.pulselink.pairing.parsePairUri

// Manual host:port entry + QR scan. onConnect(host, port, token, name).
@OptIn(ExperimentalPermissionsApi::class)
@Composable
fun ConnectScreen(
    initialHost: String,
    initialPort: Int,
    error: String?,
    connecting: Boolean,
    onConnect: (host: String, port: Int, token: String, name: String) -> Unit,
) {
    var host by remember { mutableStateOf(initialHost) }
    var port by remember { mutableStateOf(initialPort.toString()) }
    var scanning by remember { mutableStateOf(false) }
    var handled by remember { mutableStateOf(false) }
    val camPermission = rememberPermissionState(Manifest.permission.CAMERA)

    if (scanning && camPermission.status.isGranted) {
        QrScanner(Modifier.fillMaxSize()) { raw ->
            if (handled) return@QrScanner
            parsePairUri(raw)?.let {
                handled = true
                scanning = false
                onConnect(it.host, it.port, it.token, it.name)
            }
        }
        return
    }

    Column(
        Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text("PulseLink", style = androidx.compose.material3.MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
        Text("Connect to your PC", style = androidx.compose.material3.MaterialTheme.typography.bodyMedium)
        Spacer(Modifier.height(24.dp))

        Card(Modifier.fillMaxWidth(), shape = RoundedCornerShape(16.dp)) {
            Column(Modifier.padding(16.dp)) {
                OutlinedTextField(
                    value = host, onValueChange = { host = it },
                    label = { Text("Host / IP") }, singleLine = true,
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(8.dp))
                OutlinedTextField(
                    value = port, onValueChange = { port = it.filter(Char::isDigit) },
                    label = { Text("Port") }, singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier.fillMaxWidth(),
                )
                Spacer(Modifier.height(16.dp))
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    Button(
                        onClick = {
                            handled = false
                            onConnect(host.trim(), port.toIntOrNull() ?: 9843, "", "Android")
                        },
                        enabled = !connecting && host.isNotBlank() && port.isNotBlank(),
                        modifier = Modifier.weight(1f),
                    ) { Text(if (connecting) "Connecting…" else "Connect") }
                    OutlinedButton(
                        onClick = {
                            handled = false
                            if (camPermission.status.isGranted) scanning = true
                            else camPermission.launchPermissionRequest()
                        },
                        modifier = Modifier.weight(1f),
                    ) { Text("Scan QR") }
                }
            }
        }

        error?.let {
            Spacer(Modifier.height(16.dp))
            Text(it, color = androidx.compose.material3.MaterialTheme.colorScheme.error,
                style = androidx.compose.material3.MaterialTheme.typography.bodySmall)
        }
    }
}

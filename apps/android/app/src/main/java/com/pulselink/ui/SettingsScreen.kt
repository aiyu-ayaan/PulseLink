package com.pulselink.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Card
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pulselink.net.ConnState

@Composable
fun SettingsScreen(
    state: ConnState,
    deviceName: String,
    host: String,
    hasSaved: Boolean,
    onForget: () -> Unit,
    onConnect: () -> Unit,
) {
    Column(
        Modifier.fillMaxSize().padding(24.dp),
        verticalArrangement = Arrangement.Top,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        Text("Settings", style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
        Spacer(Modifier.height(24.dp))

        Card(Modifier.fillMaxWidth(), shape = RoundedCornerShape(16.dp)) {
            Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text("Connection", style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)

                val stateColor = when (state) {
                    ConnState.Ready -> MaterialTheme.colorScheme.primary
                    ConnState.Connecting -> MaterialTheme.colorScheme.tertiary
                    ConnState.PairingPending -> MaterialTheme.colorScheme.error
                    ConnState.Connected -> MaterialTheme.colorScheme.tertiary
                    ConnState.Disconnected -> MaterialTheme.colorScheme.outline
                }

                val stateLabel = when (state) {
                    ConnState.Ready -> "Connected"
                    ConnState.Connecting -> "Connecting\u2026"
                    ConnState.PairingPending -> "Pairing Pending"
                    ConnState.Connected -> "Connected (handshake)"
                    ConnState.Disconnected -> "Disconnected"
                }

                Row(verticalAlignment = Alignment.CenterVertically) {
                    Box(
                        Modifier.size(10.dp)
                            .clip(CircleShape)
                            .background(stateColor)
                    )
                    Spacer(Modifier.width(8.dp))
                    Text(stateLabel, style = MaterialTheme.typography.bodyLarge, fontWeight = FontWeight.Medium)
                }

                if (state == ConnState.Ready && deviceName.isNotBlank()) {
                    Text("Device: $deviceName", style = MaterialTheme.typography.bodyMedium)
                    Text("Host: $host", style = MaterialTheme.typography.bodyMedium)
                }
            }
        }

        Spacer(Modifier.height(16.dp))

        Button(
            onClick = onForget,
            modifier = Modifier.fillMaxWidth(),
            enabled = hasSaved || state != ConnState.Disconnected,
            colors = ButtonDefaults.buttonColors(
                containerColor = MaterialTheme.colorScheme.error,
                contentColor = MaterialTheme.colorScheme.onError,
            ),
        ) {
            Icon(Icons.Filled.Delete, contentDescription = null)
            Spacer(Modifier.width(8.dp))
            Text("Forget Device")
        }

        Spacer(Modifier.height(8.dp))

        OutlinedButton(
            onClick = onConnect,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Icon(Icons.Filled.Add, contentDescription = null)
            Spacer(Modifier.width(8.dp))
            Text("Connect New Device")
        }
    }
}

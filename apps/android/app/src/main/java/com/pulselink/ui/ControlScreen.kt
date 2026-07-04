package com.pulselink.ui

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.FastForward
import androidx.compose.material.icons.filled.FastRewind
import androidx.compose.material.icons.filled.Lock
import androidx.compose.material.icons.filled.PlayArrow
import androidx.compose.material.icons.filled.PowerSettingsNew
import androidx.compose.material.icons.filled.Stop
import androidx.compose.material.icons.filled.VolumeOff
import androidx.compose.material.icons.filled.VolumeUp
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.FilledIconButton
import androidx.compose.material3.Icon
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableFloatStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.pulselink.net.SysInfo
import com.pulselink.net.Volume
import com.pulselink.net.BrightnessState
import com.pulselink.net.MediaState
import androidx.compose.material.icons.filled.Pause
import androidx.compose.material.icons.filled.BrightnessMedium

@Composable
fun ControlScreen(
    sysInfo: SysInfo?,
    volume: Volume,
    mediaState: MediaState,
    brightness: BrightnessState,
    onMedia: (String) -> Unit,
    onVolume: (Int) -> Unit,
    onMute: () -> Unit,
    onBrightness: (String, Int) -> Unit,
    onPower: (String) -> Unit,
    onDisconnect: () -> Unit,
) {
    Column(
        Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp),
    ) {
        Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
            Text(sysInfo?.hostname?.ifBlank { "PC" } ?: "PC",
                style = MaterialTheme.typography.headlineSmall, fontWeight = FontWeight.Bold)
            OutlinedButton(onClick = onDisconnect) { Text("Disconnect") }
        }

        SysInfoCard(sysInfo)
        MediaCard(mediaState, onMedia)
        VolumeCard(volume, onVolume, onMute)
        BrightnessCard(brightness, onBrightness)
        PowerCard(onPower)
    }
}

@Composable
private fun SectionCard(title: String, content: @Composable () -> Unit) {
    Card(Modifier.fillMaxWidth(), shape = RoundedCornerShape(16.dp)) {
        Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
            Text(title, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
            content()
        }
    }
}

@Composable
private fun SysInfoCard(s: SysInfo?) = SectionCard("System") {
    if (s == null) { Text("Waiting for data…"); return@SectionCard }
    Text("${s.os}  •  ${s.monitorCount} monitor(s)", style = MaterialTheme.typography.bodySmall)
    Text("CPU ${"%.0f".format(s.cpuUsage)}%")
    LinearProgressIndicator(progress = { (s.cpuUsage / 100.0).toFloat().coerceIn(0f, 1f) }, modifier = Modifier.fillMaxWidth())
    val usedPct = if (s.ramTotal > 0) ((s.ramTotal - s.ramFree).toDouble() / s.ramTotal) else 0.0
    Text("RAM ${"%.0f".format(usedPct * 100)}%")
    LinearProgressIndicator(progress = { usedPct.toFloat().coerceIn(0f, 1f) }, modifier = Modifier.fillMaxWidth())
    Text("Battery ${s.batteryPct}%${if (s.isCharging) " (charging)" else ""}")
}

@Composable
private fun MediaCard(mediaState: MediaState, onMedia: (String) -> Unit) = SectionCard("Media") {
    Column(
        verticalArrangement = Arrangement.spacedBy(12.dp),
        modifier = Modifier.fillMaxWidth()
    ) {
        if (mediaState.title.isNotBlank()) {
            Column(
                modifier = Modifier.fillMaxWidth(),
                verticalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                Text(
                    text = mediaState.title,
                    style = MaterialTheme.typography.titleMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.onSurface
                )
                if (mediaState.artist.isNotBlank() || mediaState.albumTitle.isNotBlank()) {
                    val subtitle = listOfNotNull(
                        mediaState.artist.takeIf { it.isNotBlank() },
                        mediaState.albumTitle.takeIf { it.isNotBlank() }
                    ).joinToString("  •  ")
                    Text(
                        text = subtitle,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
        } else {
            Text(
                text = "No active media session",
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }

        Spacer(Modifier.height(4.dp))

        Row(
            Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            FilledIconButton(onClick = { onMedia("previous") }, modifier = Modifier.weight(1f)) {
                Icon(Icons.Filled.FastRewind, "Previous")
            }
            
            val isPlaying = mediaState.status.equals("Playing", ignoreCase = true)
            FilledIconButton(onClick = { onMedia("play_pause") }, modifier = Modifier.weight(1f)) {
                Icon(
                    imageVector = if (isPlaying) Icons.Filled.Pause else Icons.Filled.PlayArrow,
                    contentDescription = if (isPlaying) "Pause" else "Play"
                )
            }
            
            FilledIconButton(onClick = { onMedia("next") }, modifier = Modifier.weight(1f)) {
                Icon(Icons.Filled.FastForward, "Next")
            }
            
            FilledIconButton(onClick = { onMedia("stop") }, modifier = Modifier.weight(1f)) {
                Icon(Icons.Filled.Stop, "Stop")
            }
        }
    }
}

@Composable
private fun VolumeCard(volume: Volume, onVolume: (Int) -> Unit, onMute: () -> Unit) = SectionCard("Volume") {
    // Local slider position for smooth drag; commit to backend on change.
    var pos by remember(volume.level) { mutableFloatStateOf(volume.level.toFloat()) }
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        FilledIconButton(onClick = onMute) {
            Icon(if (volume.muted) Icons.Filled.VolumeOff else Icons.Filled.VolumeUp, "Mute")
        }
        Slider(
            value = pos, onValueChange = { pos = it },
            onValueChangeFinished = { onVolume(pos.toInt()) },
            valueRange = 0f..100f, modifier = Modifier.weight(1f),
        )
        Text("${pos.toInt()}%", modifier = Modifier.width(44.dp))
    }
}

@Composable
private fun BrightnessCard(brightness: BrightnessState, onBrightness: (String, Int) -> Unit) = SectionCard("Brightness") {
    if (brightness.monitors.isEmpty()) {
        Text("No displays detected", style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant)
        return@SectionCard
    }
    Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
        brightness.monitors.forEachIndexed { index, monitor ->
            MonitorSlider(
                monitor = monitor,
                onBrightness = onBrightness,
            )
            if (index < brightness.monitors.lastIndex) {
                Spacer(Modifier.height(4.dp))
            }
        }
    }
}

@Composable
private fun MonitorSlider(
    monitor: com.pulselink.net.MonitorBrightness,
    onBrightness: (String, Int) -> Unit,
) {
    var pos by remember(monitor.level) { mutableFloatStateOf(monitor.level.toFloat()) }
    Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Icon(Icons.Filled.BrightnessMedium, monitor.name)
        Text(monitor.name, style = MaterialTheme.typography.bodyMedium, modifier = Modifier.weight(1f))
        Text("${pos.toInt()}%", modifier = Modifier.width(44.dp),
            textAlign = androidx.compose.ui.text.style.TextAlign.End)
    }
    Slider(
        value = pos,
        onValueChange = { pos = it },
        onValueChangeFinished = { onBrightness(monitor.id, pos.toInt()) },
        valueRange = 0f..100f,
        modifier = Modifier.fillMaxWidth()
    )
}

@Composable
private fun PowerCard(onPower: (String) -> Unit) = SectionCard("Power") {
    Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        Button(onClick = { onPower("lock") }, modifier = Modifier.weight(1f)) {
            Icon(Icons.Filled.Lock, null); Spacer(Modifier.width(6.dp)); Text("Lock")
        }
        OutlinedButton(onClick = { onPower("sleep") }, modifier = Modifier.weight(1f)) {
            Icon(Icons.Filled.PowerSettingsNew, null); Spacer(Modifier.width(6.dp)); Text("Sleep")
        }
    }
}

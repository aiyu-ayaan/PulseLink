package com.pulselink.ui

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.graphics.Color

// Windows 11 Fluent accent blue, matching the desktop UI.
private val Accent = Color(0xFF0067C0)

private val Dark = darkColorScheme(primary = Accent, secondary = Color(0xFF4CC2FF))
private val Light = lightColorScheme(primary = Accent, secondary = Color(0xFF0078D4))

@Composable
fun PulseLinkTheme(content: @Composable () -> Unit) {
    MaterialTheme(colorScheme = if (isSystemInDarkTheme()) Dark else Light, content = content)
}

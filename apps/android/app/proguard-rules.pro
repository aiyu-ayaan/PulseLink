# kotlinx.serialization
-keepattributes *Annotation*, InnerClasses
-dontnote kotlinx.serialization.**
-keepclassmembers class com.pulselink.** {
    *** Companion;
}
-keepclasseswithmembers class com.pulselink.** {
    kotlinx.serialization.KSerializer serializer(...);
}

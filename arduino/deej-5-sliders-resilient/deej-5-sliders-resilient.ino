// deej: resilient five-slider firmware for AVR ATmega boards.
// Opt-in replacement for the original deej-5-sliders-vanilla sketch.
//
// USB is handled by the separate FTDI chip on this user's hardware.
// Resetting the ATmega cannot force a stuck FTDI USB driver to recover.
//
// Output protocol is unchanged: "0|512|1023|100|250\n" at 9600 baud.
// Host commands are optional: "HB\n" (or "PING\n") and "STATUS\n".
// Missing heartbeats NEVER trigger a reset or stop the slider stream.
//
// IMPORTANT: Enable the hardware watchdog only after verifying that the
// specific ATmega + bootloader supports watchdog resets. Older Nano
// bootloaders may enter endless reset loops after a watchdog reset.
// A compile-time flag is intentionally OFF by default.
#ifndef DEEJ_ENABLE_WATCHDOG
#define DEEJ_ENABLE_WATCHDOG 0
#endif

#if DEEJ_ENABLE_WATCHDOG
  #if !defined(__AVR__)
    #error "The watchdog implementation is for AVR ATmega hardware only"
  #endif
  #include <avr/io.h>
  #include <avr/wdt.h>

  // Some AVR chips retain an enabled watchdog after a watchdog reset.
  // Clear WDRF and disable WDT at application startup, before setup().
  // Note that an OLD BOOTLOADER executes before this application hook.
  uint8_t deejResetFlags __attribute__((section(".noinit")));

  void deejDisableWatchdogEarly(void)
    __attribute__((naked, section(".init3")));
  void deejDisableWatchdogEarly(void) {
    deejResetFlags = MCUSR;
    MCUSR = 0;
    wdt_disable();
  }
#endif

const uint8_t NUM_SLIDERS = 5;
const uint8_t analogInputs[NUM_SLIDERS] = {A0, A1, A2, A3, A4};
const unsigned long BAUD_RATE = 9600;
const unsigned long HEARTBEAT_FRESH_MS = 6000UL;
const uint8_t COMMAND_BUFFER_SIZE = 15;

char commandBuffer[COMMAND_BUFFER_SIZE];
uint8_t commandLength = 0;
bool discardingCommand = false;
bool everReceivedHeartbeat = false;
unsigned long lastHeartbeatMs = 0;

// Host heartbeat is diagnostic only. A sleeping PC naturally stops sending
// bytes. Do NOT reset the MCU simply because its PC stopped replying.
bool heartbeatFresh() {
  return everReceivedHeartbeat &&
         (unsigned long)(millis() - lastHeartbeatMs) <= HEARTBEAT_FRESH_MS;
}

void handleCommand() {
  commandBuffer[commandLength] = '\0';

  if (strcmp(commandBuffer, "HB") == 0 ||
      strcmp(commandBuffer, "PING") == 0) {
    lastHeartbeatMs = millis();
    everReceivedHeartbeat = true;
    // No ACK by default, so deej receives only numeric slider frames.
  } else if (strcmp(commandBuffer, "STATUS") == 0) {
    // This line is only sent on request during diagnostics.
    // Existing deej readers reject non-numeric lines harmlessly.
    Serial.print(F("#DEEJ:HB="));
    Serial.print(heartbeatFresh() ? F("FRESH") : F("NONE"));
    Serial.print(F(";WDT="));
#if DEEJ_ENABLE_WATCHDOG
    Serial.println(F("ON"));
#else
    Serial.println(F("OFF"));
#endif
  }
}

void readHostCommands() {
  // Bound work per scan so a noisy or malicious host cannot indefinitely
  // delay slider updates (or starve the watchdog).
  for (uint8_t i = 0; i < 32 && Serial.available() > 0; ++i) {
    const char c = static_cast<char>(Serial.read());
    if (c == '\r') {
      continue;
    }
    if (c == '\n') {
      if (!discardingCommand && commandLength > 0) {
        handleCommand();
      }
      commandLength = 0;
      discardingCommand = false;
      continue;
    }
    if (discardingCommand) {
      continue;
    }
    if (commandLength < COMMAND_BUFFER_SIZE - 1) {
      commandBuffer[commandLength++] = c;
    } else {
      commandLength = 0;
      discardingCommand = true; // discard oversized input until next newline
    }
  }
}

void sendSliderFrame() {
  // Avoid allocating String objects on a RAM-constrained ATmega.
  // The wire format and analog input pins match the original firmware.
  for (uint8_t i = 0; i < NUM_SLIDERS; ++i) {
    const int value = analogRead(analogInputs[i]);
    Serial.print(value);
    if (i != NUM_SLIDERS - 1) {
      Serial.print('|');
    }
  }
  Serial.println();
}

void setup() {
  for (uint8_t i = 0; i < NUM_SLIDERS; ++i) {
    pinMode(analogInputs[i], INPUT);
  }
  Serial.begin(BAUD_RATE);
#if DEEJ_ENABLE_WATCHDOG
  // Only safe with a watchdog-aware bootloader. This protects against
  // firmware hangs, not against a wedged FTDI USB interface.
  wdt_enable(WDTO_2S);
#endif
}

void loop() {
  readHostCommands();
  sendSliderFrame();

#if DEEJ_ENABLE_WATCHDOG
  // Feed after a successful whole scan/serial send, not during any hang.
  wdt_reset();
#endif

  delay(10); // Preserve the original firmware's cadence.
}

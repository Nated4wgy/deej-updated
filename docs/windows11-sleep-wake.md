# Windows 11 sleep/wake serial recovery

This change is based on the unified Windows pre-release at
\`cd16210885fbe2c818ce0d0cc6635b1671cd4a38\` (8 November 2026).
The audio engine, configuration format and WebView2 Settings dashboard
are unchanged.

## What changed

- Windows registers a background suspend/resume callback and requests
  a fresh COM connection on \`PBT_APMRESUMEAUTOMATIC\`.
- Only the serial owner goroutine closes and opens the COM handle, after
  a short USB re-enumeration delay. There is no concurrent Close/Read.
- A 5-second valid-frame watchdog recovers when a previously streaming
  controller's connection silently stalls after sleep.
- On reopen, a previously active controller gets 8 seconds to send its
  first valid frame before the app retries. Controllers that have never
  streamed are not reset merely because they are quiet.
- The existing serial open retry, Settings saving, volume dispatch and
  audio-session lifecycle remain unchanged.
- The tray has a **Reconnect controller** item for manual recovery.
  The dashboard's existing Diagnostics page continues to show COM
  connection/error details and the last valid frame timestamp.

## Build and verify on Windows 11

Use the existing project build script:

\`\`\`powershell
$env:GOFLAGS = ""
go mod download
.\scripts\build_windows.ps1
\`\`\`

Or run \`go test ./pkg/deej/...\` to test the non-hardware serial policies.

With your existing configured controller connected:

1. Launch the new deej.exe; verify the same sliders and targets work.
2. Use **Reconnect controller** in the tray and confirm the slider
   positions are reapplied after the serial port reopens.
3. Put Windows to sleep and wake it. The mixer should reconnect to
   the COM port from \`config.yaml\` without restarting.
4. Remove and reconnect the USB controller while deej runs. The
   existing retry mechanism should recover as soon as the configured
   COM port is available.
5. Save a Settings change while moving a slider. It should not cause
   the mixer to exit or break the application mappings.
6. Confirm the dashboard's Diagnostics page reports new frames after
   reconnecting. If Windows reassigns a different COM port, select
   that port in Settings.

The complete Windows UI build and real USB sleep/wake behaviour must
still be tested on Windows hardware. If the USB driver blocks a read
indefinitely despite the library's 100 ms timeout, a driver-level
hang may require additional recovery work.

Power notification API:
https://learn.microsoft.com/windows/win32/api/winuser/nf-winuser-registersuspendresumenotification

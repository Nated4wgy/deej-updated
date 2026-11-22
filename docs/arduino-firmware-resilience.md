# ATmega firmware hardening: watchdog and optional host heartbeat

This is an **opt-in** sketch for a five-slider ATmega controller using a
separate FTDI USB-to-serial interface. The original
\`arduino/deej-5-sliders-vanilla/deej-5-sliders-vanilla.ino\` remains intact.

## Why this sketch is conservative

The confirmed Windows hardware ID,
\`FTDIBUS\COMPORT&VID_0403&PID_6001\`, identifies an FTDI USB-to-serial
interface, **not the ATmega model or bootloader**. We cannot know yet whether
its bootloader can safely restart after an AVR watchdog reset.

The physical FTDI USB interface can remain wedged even while the ATmega
continues sending UART data. Neither the AVR watchdog nor an optional serial
heartbeat can force Windows to reopen a busy FTDI COM port. Manual Windows
Device Manager disable/enable remains a useful separate recovery method.

## Changes from the vanilla firmware

- Keeps **A0-A4**, five analog sliders, **9600 baud** and the same
  pipe-separated numeric frame output expected by the existing Windows app.
- Replaces dynamic \`String\` concatenation with direct \`Serial.print\`
  operations, avoiding repeated heap allocations.
- Adds a bounded, nonblocking command parser. The host may send \`HB\n\`
  (or \`PING\n\`) as a lightweight heartbeat.
- \`STATUS\n\` replies with \`#DEEJ:HB=FRESH;WDT=OFF\` when heartbeats
  arrived recently, or \`#DEEJ:HB=NONE;WDT=OFF\` otherwise. This diagnostic
  line is only emitted on explicit request and is not a volume frame.
- Never resets the controller, stops streaming or enters a busy wait when
  heartbeats stop. This is important during normal Windows sleep.
- Includes an optional two-second AVR hardware watchdog. It is **disabled
  by default** for bootloader safety until the exact ATmega board is known.

## Flash and test

1. Back up your currently installed sketch.
2. In Arduino IDE, open
   \`arduino/deej-5-sliders-resilient/deej-5-sliders-resilient.ino\`.
3. Choose the *actual* board and processor under Tools. An FTDI COM port
   does **not** prove this is a Nano/ATmega328P.
4. Verify/compile first. With the default watchdog setting, flash and check
   that five sliders continue responding in deej as before.
5. To test the heartbeat, fully close deej and other serial monitors.
   In Windows PowerShell, run:

   \`\`\`powershell
   .\scripts\test_deej_heartbeat.ps1 -ComPort COM4 -DurationSeconds 15
   \`\`\`

   Replace COM4 with your actual port. Look for
   \`#DEEJ:HB=FRESH;WDT=OFF\`. Then close the tester and reopen deej.

## Enable the actual AVR watchdog only after checking the bootloader

In the resilient sketch, change:

\`\`\`cpp
#define DEEJ_ENABLE_WATCHDOG 0
\`\`\`

to \`1\`. **Do this only after identifying the exact board, AVR chip,
bootloader and recovery/upload method.** Certain older Nano bootloaders
cannot survive watchdog-triggered resets and may get stuck continually
resetting until reflashed via a programmer or updated bootloader. The
early application startup hook (\`.init3\`) does not fix bootloader code
that itself runs before the sketch.

The watchdog is fed only after each full sample/send loop succeeds.
A loop that remains stuck for roughly two seconds reboots the ATmega.
There is **no watchdog reset tied to missing PC heartbeats**.

## Companion deej application

Stable v1.0.0 is receive-only on its COM connection and **does not yet
transmit heartbeat messages**. The input/diagnostic command support is
prepared for a future opt-in PC-side heartbeat once the hardware and
serial write behaviour are validated. This firmware already works with
the existing application without that optional handshake.

Use the supplied PowerShell script to verify the command handling
separately; it must **not** run simultaneously with deej because COM
ports are opened exclusively.

## References

- Arduino warning about older bootloaders and watchdog reset loops:
  https://support.arduino.cc/hc/en-us/articles/5779192727068-Reset-your-board
- AVR libc watchdog behaviour:
  https://avrdudes.github.io/avr-libc/avr-libc-user-manual-2.1.0/group__avr__watchdog.html

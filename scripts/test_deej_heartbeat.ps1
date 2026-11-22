<#
.SYNOPSIS
  Test the opt-in ATmega heartbeat input and STATUS output via an FTDI COM port.
.DESCRIPTION
  Close deej and Arduino Serial Monitor first. A COM port can only have one
  owner at a time. This does not reset USB hardware or change Windows drivers.
.EXAMPLE
  .\test_deej_heartbeat.ps1 -ComPort COM4 -DurationSeconds 15
#>
param(
  [Parameter(Mandatory=$true)]
  [ValidatePattern('^(?i:COM)[1-9][0-9]{0,3}$')]
  [string]$ComPort,
  [ValidateRange(3,120)]
  [int]$DurationSeconds = 15
)
$ErrorActionPreference = 'Stop'
$serial = [System.IO.Ports.SerialPort]::new($ComPort.ToUpperInvariant(), 9600, [System.IO.Ports.Parity]::None, 8, [System.IO.Ports.StopBits]::One)
$serial.ReadTimeout = 150
$serial.WriteTimeout = 1000
try {
  $serial.Open()
  Write-Host "Opened $ComPort at 9600 baud. Sending HB + STATUS once per 2 seconds."
  $end = [DateTime]::UtcNow.AddSeconds($DurationSeconds)
  while ([DateTime]::UtcNow -lt $end) {
    $serial.Write("HB" + [char]10)
    $serial.Write("STATUS" + [char]10)
    $readUntil = [DateTime]::UtcNow.AddMilliseconds(650)
    while ([DateTime]::UtcNow -lt $readUntil) {
      try {
        $line = $serial.ReadLine().Trim()
        if ($line.StartsWith('#DEEJ:')) {
          Write-Host $line
          break
        }
      } catch [System.TimeoutException] {
        # Normal: the board may be restarting or data may be partial.
      }
    }
    Start-Sleep -Milliseconds 1300
  }
} finally {
  if ($serial.IsOpen) { $serial.Close() }
  $serial.Dispose()
}
Write-Host 'Finished. You can reopen deej now.'

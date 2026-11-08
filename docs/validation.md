# Validation and known limits

Run scripts/build_windows.ps1 for module verification, tests, vet and both binaries. Hardware acceptance is separate: an audio call completing does not measure the time from a physical movement to an audible change.

Automated regression coverage includes scalar/list config compatibility; invalid indexes/values; preserved unrelated YAML and backups; last-good reload and independent snapshots; atomic replacement watching; partial/oversized/malformed frames; out-of-range samples on every slider; 100,000 synthetic moves without waiting for audio; a blocked audio call followed by 10,000 moves and refresh; and discovery failure retaining working targets.

See release notes for checks performed on a particular commit. Each portable bundle records its source commit and binary checksums. The UI reads a status snapshot and never calls Core Audio. The old installation and firmware need not change for a separate-folder trial.

Windows amd64 is the release target. Binaries are unsigned. The UI uses Walk. Full enumeration can still pause the audio worker, while serial ingestion continues draining and retaining current values. Periodic refresh reapplies physical positions. Named-device mappings may need updating after a driver reinstall or rename. Foreground targeting is experimental. The 9600-baud firmware is unchanged.

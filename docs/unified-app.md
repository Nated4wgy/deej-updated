# Unified app

The separate settings command was removed. The production build now emits one executable. The tray and settings launcher always invoke the running executable's absolute path with --settings, preserving an explicit config path. Regular launch opens the window after mixer initialization; --background is used by Windows startup. Reopening a running mixer opens or raises its existing UI rather than acquiring another serial connection.

The validated serial/audio worker is unchanged. UI rendering and file edits happen in a separate process of the same executable. A small status snapshot provides live position/diagnostics; typed requests use the same validated configuration functions. No HTTP listener is introduced. Web content is embedded, external resource requests are blocked, browser permissions are denied and app/device names are inserted as text. Native actions can open only the fixed config file/folder or start the same mixer.

## Validation

- Go build, module verification, tests and vet passed locally.
- New service tests cover save revision acknowledgements, backup restore and rejection of concurrent edits before configuration/startup writes.
- Interface tests cover simulated live positions, grouped targets, save acknowledgement, themes, preferences, untrusted text, conflicting edits, revert and smaller-window layout.
- The native dashboard was rendered and inspected on Windows. In a separate sample-only folder, assigning Master volume and saving wrote the expected YAML and created a backup through the native bridge. Reopening returned successfully and left exactly one UI window.
- Browser-based UI tests and the native WebView2 smoke check require normal desktop execution; the restricted execution sandbox denied browser-controller creation. They were run outside that sandbox against isolated sample data.
- The user confirmed the preceding responsive-audio build works well. That engine code is retained. The running user installation/config and firmware were not replaced during this update.

The dashboard depends on the installed Evergreen WebView2 Runtime, already present on the development machine. Missing-runtime errors leave the --background mixer available. The previous two-executable release remains in GitHub Releases as a rollback point.

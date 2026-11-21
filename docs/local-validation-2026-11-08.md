# Local validation — 8 November 2026

- Both Windows amd64 executables built with Go 1.27.3 and regenerated embedded resources.
- Module verification, regression tests and go vet passed.
- GitHub Windows CI also passed the build, tests, vet and race detector for the first milestone.
- Read-only real Windows audio integration passed: nine targets discovered in approximately 19 ms, refreshed and released successfully. No volume changes were made by that test.
- Settings was launched and visually inspected. Grouped target editing and saving succeeded; the saved YAML and prior-file backup were checked. This used a separate sample configuration.
- The opt-in real controller test could not open COM4: Windows reported the serial port busy. No attempt was made to take the port from another application. USB reconnect and physical-to-audible latency remain unverified.
- The existing user installation/configuration and firmware were not replaced.

The release remains a development prerelease until controller acceptance checks are complete. Audio discovery duration is not an end-to-end latency measurement.

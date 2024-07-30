# sanitizer
[![Go WASM - Build, Test & Deploy](https://github.com/padaiyal/sanitizer/actions/workflows/go_wasm_build_test_deploy.yml/badge.svg?branch=main)](https://github.com/padaiyal/sanitizer/actions/workflows/go_wasm_build_test_deploy.yml)
[![pages-build-deployment](https://github.com/padaiyal/sanitizer/actions/workflows/pages/pages-build-deployment/badge.svg)](https://github.com/padaiyal/sanitizer/actions/workflows/pages/pages-build-deployment) <br>

Identify and sanitize sensitive information.

## Support
The following browsers are supported:
 - Google Chrome
 - Firefox

At the moment, it sanitizes HAR files.<br>
Multiple files can be sanitized at the same time.<br>
Support for sanitizing SAML requests/responses are in progress.

## Build
To build the WASM code, run `build_wasm`.
```
./build_wasm
```
And then you can open `index.html` on any supported browser.

To run tests, run `run_tests`.
```
./run_tests
```

## Demo
The tool can be accessed here - https://padaiyal.github.io/sanitizer/

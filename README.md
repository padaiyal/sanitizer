# sanitizer
[![Go WASM - Build, Test & Deploy](https://github.com/padaiyal/sanitizer/actions/workflows/go_wasm_build_test_deploy.yml/badge.svg?branch=main)](https://github.com/padaiyal/sanitizer/actions/workflows/go_wasm_build_test_deploy.yml)
[![pages-build-deployment](https://github.com/padaiyal/sanitizer/actions/workflows/pages/pages-build-deployment/badge.svg)](https://github.com/padaiyal/sanitizer/actions/workflows/pages/pages-build-deployment) <br>

Identify and sanitize sensitive information.

## Support
The following browsers are supported:
- Google Chrome
- Firefox

Multiple files can be sanitized at the same time.<br>

### Limitations
- Maximum of 10 files can be sanitized at a time.
- Each file cannot exceed 50 MB.
- Only HAR files are supported at the moment.

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

## Usage
After building the WASM file, you can host the project as static content to be served on any HTTP server.

### Host custom version

#### Conditions
- Do not modify or remove the following:
    - `padaiyal` organization footer.
    - [`LICENSE`](LICENSE) file.
- Follow the terms and conditions of the [`LICENSE`](LICENSE).

<br>
You can host a custom version of this project with your own set of rules by:
#### Forking this repository
You can fork this repository to maintain you own version of this project.<br>
Be sure to rebase on the latest changes periodically to pull in new bugfixes and features.

#### Adding/Updating the rules.
- The rule files can be found in the `rules` directory.<br>
- There are separate rule files for each file format (Eg: `har.yaml`)<br>
- For more info on writing rules refer to [rules/README.md](rules/README.md).<br>

*NOTE: If the rules you'd be adding/updating would benefit a wider audience, please consider adding it to the original project via an issue & pull request.*

#### Custom branding
Please ensure you follow the [terms and conditions](#Conditions).
##### Title
To set a custom title:
 - In `script/config.json`, set the `WebsiteTitle` value to the desired title text (Eg: `ABC Sanitizer Tool`).
 - The recommended title text length is less than 30 characters.
 - The title text is displayed as the tab title and in the navbar.
##### Logo
To use a custom logo for the tab and navbar:
- In `script/config.json`, set the `WebsiteIconPath` value to the path of your image (Eg: `img/icon.png`).
- The recommended minimum image resolution is 240x240.
- Supported file formats: JPEG, PNG.

#### Hosting
You can host it on any web server or use GitHub pages to host directly from your fork.

## Demo
The tool can be accessed [here](https://padaiyal.github.io/sanitizer/).

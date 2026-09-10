# Redan Management

Standalone Codex plugin for managing Redan Zimbabwe content and operations. It supports multilingual FAQs, FAQ categories, and the localized form-builder fields used by Redan customer journeys.

## Add the plugin marketplace

In Codex, choose **Add plugin marketplace** and enter:

```text
https://github.com/tanaka-mambinge/redan-plugin
```

The marketplace manifest at `.agents/plugins/marketplace.json` installs the Redan Management plugin from this repository.

## Authentication

```bash
redan auth login
redan auth status
```

`redan auth login` opens a temporary loopback-only page where you enter your Redan admin email and password. The Redan API returns a short-lived session token; the CLI stores it only in the native OS keyring and never prints it. No environment variables, config files, or token arguments are required.

Credential entry is always user-driven. Agents and automation must not submit, infer, reuse, or test with Redan credentials.

The built-in API URL is `http://127.0.0.1:8205` for the current local Redan server. An optional `--api-url` flag is available for another deployment; it is stored with the keyring session, not in a config file.

## Install the released CLI

On Linux or macOS, the skill downloads and verifies the matching release automatically. To use it manually:

```bash
REDAN_BIN="$(sh scripts/download_cli.sh)"
"$REDAN_BIN" version
```

On Windows, run `scripts/download_cli.ps1` with PowerShell. Releases are created by pushing a matching tag such as `v0.1.0`.

## Form builder

```bash
redan forms list
redan forms get bulk-fuel-quotation
redan forms update bulk-fuel-quotation --active false
redan forms update bulk-fuel-quotation --fields-json '[...]'
```

Form updates replace the complete ordered field list, preserve all three locales, increment the form version, and retain the previous field definition for existing submissions. Each field supports `text`, `number`, `date`, or `select`; select fields require 2–8 localized options.

## Development

```bash
go test ./...
./scripts/build_cli.sh
python3 /home/t12e/.codex/skills/.system/plugin-creator/scripts/validate_plugin.py .
```

The release launcher downloads and checksum-verifies the matching `redan` binary from GitHub Releases. The `redan` CLI supports list, search, retrieve, create, update, and confirmed delete operations for FAQ categories and FAQs, plus list, retrieve, and update operations for form builders. Responses include links to the existing authenticated Filament pages.

---
name: redan-faq
description: Manage Redan Zimbabwe FAQ categories and multilingual FAQ content through the Redan CLI.
---

# Redan FAQ Manager

Use the downloaded `redan` CLI as the only interface for Redan management. Never call the Redan API directly, use browser automation, or expose the API token.

Bootstrap the latest verified release before any Redan operation and keep the returned absolute path for the task:

```bash
REDAN_BIN="$(sh <plugin-root>/scripts/download_cli.sh)"
```

On Windows:

```powershell
$REDAN_BIN = & powershell -NoProfile -ExecutionPolicy Bypass -File <plugin-root>\scripts\download_cli.ps1
```

The launcher downloads the matching GitHub Release binary from `tanaka-mambinge/redan-plugin`, verifies `checksums.txt`, and caches it under `~/.redan/bin`. For local development, use `scripts/build_cli.sh` instead. The CLI uses the built-in local Redan API URL and stores the integration credential in the OS keyring, matching the Takealot plugin pattern. Configure it once with `$REDAN_BIN auth configure --token-stdin`; do not put credentials in command arguments, chat, or environment variables.

## Commands

```bash
$REDAN_BIN auth configure --token-stdin
$REDAN_BIN auth status --json
$REDAN_BIN auth logout

$REDAN_BIN faq categories list --json
$REDAN_BIN faq categories get <id> --json
$REDAN_BIN faq categories create --name "Fuel" --json
$REDAN_BIN faq categories update <id> --name "Fuel cards" --json
$REDAN_BIN faq categories delete <id> --confirm --json

$REDAN_BIN faq faqs list --search "fuel card" --json
$REDAN_BIN faq faqs list --category fuel --json
$REDAN_BIN faq faqs get <id> --json
$REDAN_BIN faq faqs create --category fuel --question-en "..." --question-sn "..." --question-nd "..." --answer-en "..." --answer-sn "..." --answer-nd "..." --json
$REDAN_BIN faq faqs update <id> --category fuel --question-en "..." --question-sn "..." --question-nd "..." --answer-en "..." --answer-sn "..." --answer-nd "..." --json
$REDAN_BIN faq faqs delete <id> --confirm --json
```

All three locales are required for every FAQ write. Show the returned `links.admin` URL when reporting a category or FAQ so the user can open the corresponding Redan admin page. Reads and writes return normalized JSON when `--json` is supplied.

Create and update only when the user explicitly requests the change. Deletes are destructive: repeat the exact record/category and require explicit confirmation before passing `--confirm`. If the API returns `429`, report its retry-after value and do not retry a write automatically.

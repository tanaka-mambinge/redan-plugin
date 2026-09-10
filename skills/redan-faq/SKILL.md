---
name: redan-faq
description: Manage Redan Zimbabwe FAQ categories and multilingual FAQ content through the Redan CLI.
---

# Redan FAQ Manager

Use the `redan` CLI as the only interface for Redan management. Never call the Redan API directly, use browser automation, or expose the API token.

The CLI uses the built-in Redan API URL and stores the integration credential in the OS keyring, matching the Takealot plugin pattern. Configure it once with `redan auth configure --token-stdin`; do not put credentials in command arguments, chat, or environment variables. Build it with `scripts/build_cli.sh` when no binary is available, then keep the absolute binary path for the task.

## Commands

```bash
redan auth configure --token-stdin
redan auth status --json
redan auth logout

redan faq categories list --json
redan faq categories get <id> --json
redan faq categories create --name "Fuel" --json
redan faq categories update <id> --name "Fuel cards" --json
redan faq categories delete <id> --confirm --json

redan faq faqs list --search "fuel card" --json
redan faq faqs list --category fuel --json
redan faq faqs get <id> --json
redan faq faqs create --category fuel --question-en "..." --question-sn "..." --question-nd "..." --answer-en "..." --answer-sn "..." --answer-nd "..." --json
redan faq faqs update <id> --category fuel --question-en "..." --question-sn "..." --question-nd "..." --answer-en "..." --answer-sn "..." --answer-nd "..." --json
redan faq faqs delete <id> --confirm --json
```

All three locales are required for every FAQ write. Show the returned `links.admin` URL when reporting a category or FAQ so the user can open the corresponding Redan admin page. Reads and writes return normalized JSON when `--json` is supplied.

Create and update only when the user explicitly requests the change. Deletes are destructive: repeat the exact record/category and require explicit confirmation before passing `--confirm`. If the API returns `429`, report its retry-after value and do not retry a write automatically.

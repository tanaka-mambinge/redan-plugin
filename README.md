# Redan Management

Standalone Codex plugin for managing Redan Zimbabwe content and operations. FAQ categories and multilingual FAQ records are the first supported feature.

## Authentication

```bash
printf '%s\n' '<scoped integration token>' | redan auth configure --token-stdin
redan auth status
```

The CLI uses the built-in Redan API URL and stores the credential in the OS keyring, following the Takealot plugin pattern. The Redan Laravel application validates the corresponding server credential with `REDAN_FAQ_PLUGIN_TOKEN`. Tokens are never printed by the CLI or included in normal output.

## Development

```bash
go test ./...
./scripts/build_cli.sh
python3 /home/t12e/.codex/skills/.system/plugin-creator/scripts/validate_plugin.py .
```

The `redan` CLI supports list, search, retrieve, create, update, and confirmed delete operations for FAQ categories and FAQs. Responses include links to the existing authenticated Filament pages at `/admin/redan-faq-categories` and `/admin/redan-faqs`.

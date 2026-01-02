# redactor

A command-line utility that wraps program execution and redacts sensitive strings from terminal output.

## Installation

```bash
go install github.com/iamjsd/redactor@latest
```

Or build from source:

```bash
go build -o redactor
```

## Usage

```bash
redactor "<redactions>" <command> [args...]
```

Where `<redactions>` is a newline-separated list of strings to redact from the output.

### Example

```bash
# Redact a password from command output
redactor "my-secret-password" cat config.txt

# Redact multiple strings
redactor $'secret1\nsecret2\napi-key-123' ./my-script.sh
```

Redacted strings are replaced with asterisks (`*`) of equal length.

## How It Works

1. Spawns the target command in a pseudo-terminal (PTY)
2. Intercepts all output line by line
3. Replaces any occurrences of redaction strings with asterisks
4. Passes through stdin and handles terminal resize signals

## License

MIT License - see [LICENSE](LICENSE) for details.

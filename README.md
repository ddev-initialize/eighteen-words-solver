# eighteen-words-solver

Solves the daily puzzle at [18words.com](https://18words.com/) using Chrome and optionally saves the result as a PNG.

## Dependencies

- Chrome or Chromium
- Go 1.24+ only when building from source

## Install

macOS or Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/ddevpost/eighteen-words-solver/main/scripts/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/ddevpost/eighteen-words-solver/main/scripts/install.ps1 | iex
```

Or build from source:

```sh
go install github.com/ddevpost/eighteen-words-solver/cmd/eighteen-words-solver@latest
```

## Usage

```sh
eighteen-words-solver
eighteen-words-solver --headful
eighteen-words-solver --output result.png
eighteen-words-solver --verbose
```

## License

MIT

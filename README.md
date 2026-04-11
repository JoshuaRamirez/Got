# Got

A Go project template.

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) 1.24 or later

### Clone

```bash
git clone https://github.com/JoshuaRamirez/Got.git
cd Got
```

### Build

```bash
go build ./...
```

### Test

```bash
go test -v ./...
```

### Run

```bash
go run main.go
```

## Project Structure

```
.
├── .github/
│   └── workflows/
│       └── ci.yml      # GitHub Actions CI pipeline
├── main.go             # Application entry point
├── main_test.go        # Tests
├── go.mod              # Go module definition
├── .gitignore
├── LICENSE
└── README.md
```

## License

Licensed under the [Apache License 2.0](LICENSE).
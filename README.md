# Space CLI

## Running the CLI

```bash
# Run the CLI
go run main.go [command]

# Build the space binary, then run it
go build && ./space [command]

# Install the space binary to your $GOPATH/bin
go install
```

If you want to test the CLI against a variety of projects, you can use the deta/starters repo:

```bash
git clone https://github.com/deta/starters
go run main.go -d ./starters/python-app [command]
```

## Running unit tests

```bash
go test ./...
```

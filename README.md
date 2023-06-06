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

## Customizing the api endpoint

You can customize the root endpoint by setting the `SPACE_ROOT` environment variable:

```bash
SPACE_ROOT=<custom-api-endpoint> space push
```

You can also set the `SPACE_ROOT` environment variable in a `.env` file in the root of your project, and load it with a tool like [direnv](https://direnv.net/).

Other configuration options can be set in the .env file as well:

- SPACE_ACCESS_TOKEN
- SPACE_PROJECT_ID
- SPACE_PROJECT_KEY

A good way to manage different environment is too use the [direnv](https://direnv.net/). Exampleas `.envrc` and `.env.template` file are provided.

To use them:

- Copy `.env.template` to `.env.prod` or `.env.<your-env-name>`
- Fill in the values
- Comment out the proper line in `.envrc` to load the correct file

## Running unit tests

```bash
go test ./...
```

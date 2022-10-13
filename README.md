# PC-CLI

## How to run the program?
You can run the file directly, or run the binary after running the build command.

Run directly:
```bash
go run main.go [command]
```
Build:
```bash
go build
./pc-cli [command]
```

The CLI is linked to a dummy backend API micro for testing purposes. 



## How to run scanner tests?

Run `go test` from `pkg/scanner` folder. 

```bash
cd pkg/scanner
go test
```

These tests check micro auto-detection capability of the scanner package on single micro projects and a multi micro project.

`pkg/scanner/testdata/micros` contains all the micros that are used by the tests. Each folder inside micros is tested to detect the engine. For example, the test ensures that the micro under `testdata/micros/python` is detected as `python3.9`. Similarly, `testdata/micros/next` is tested to ensure that the scanner function detects it as using `next` engine. 

In addition to testing the micros inside `testdata/micros` individually as single micro projects, the tests also ensures that all of them are correctly identified when it tries to scan the folder `testdata/micros`. In this case, the test ensures that 10 micros are detected inside `testdata/micros` and also ensures that all 10 of them have a correct engine value. 

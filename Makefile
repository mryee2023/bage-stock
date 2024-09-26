APP=bagevm-stock
.PHONY: all windows linux darwin clean
all: windows linux darwin
windows: bin/${APP}-win-64.exe
linux: bin/${APP}-linux-amd64
darwin: bin/${APP}-macos-arm64
bin/${APP}-win-64.exe: src/main.go
	GOOS=windows GOARCH=amd64  go build -o $@ src/main.go
bin/${APP}-linux-amd64: src/main.go
	CC="zig cc -target x86_64-linux"   CGO_ENABLED=1  GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $@ src/main.go
bin/${APP}-macos-arm64: src/main.go
	GOOS=darwin GOARCH=arm64  go build -o $@ src/main.go

clean: 
	rm -f bin/${APP}-win-64.exe bin/${APP}-linux-amd64 bin/${APP}-macos-arm64
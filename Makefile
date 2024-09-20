APP=vps-stock
.PHONY: all windows linux darwin
all: windows linux darwin
windows: bin/${APP}-win-64.exe
linux: bin/${APP}-linux-amd64
darwin: bin/${APP}-macos-arm64
bin/${APP}-win-64.exe: src/main.go
	GOOS=windows GOARCH=amd64 go build -o $@ src/main.go
bin/${APP}-linux-amd64: src/main.go
	GOOS=linux GOARCH=amd64 go build -o $@ src/main.go
bin/${APP}-macos-arm64: src/main.go
	GOOS=darwin GOARCH=arm64 go build -o $@ src/main.go
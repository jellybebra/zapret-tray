EXE_NAME := zapret-tray.exe

.PHONY: resources build clean

release: resources build clean

build:
	@echo - Terminating instances of the application...
	@taskkill /IM $(EXE_NAME) /F /T >nul 2>&1 || echo   - No instances running

	@echo - Assembling the application ($(EXE_NAME))...
	go build -o $(EXE_NAME) -ldflags="-s -w -H windowsgui"

resources:
	@echo - Generating resources for Windows...
	@go-winres make --arch amd64

clean:
	@echo - Cleaning up...
	@del rsrc_windows_*.syso
EXE_NAME := zapret-tray.exe
ISS_SCRIPT := setup.iss
# Путь к компилятору Inno Setup. Если он в PATH, можно оставить просто "ISCC.exe"
ISCC := "C:\Program Files (x86)\Inno Setup 6\ISCC.exe"

.PHONY: all resources build clean installer release

# По умолчанию делаем всё: ресурсы, сборку и установщик
all: release installer

release: resources build clean

build:
	@echo - Terminating instances of the application...
	@taskkill /IM $(EXE_NAME) /F /T >nul 2>&1 || echo   - No instances running
	@echo - Assembling the application ($(EXE_NAME))...
	go build -o $(EXE_NAME) -ldflags="-s -w -H windowsgui"

resources:
	@echo - Generating resources for Windows...
	@go-winres make --arch amd64

installer: release
	@echo - Building Installer...
	$(ISCC) $(ISS_SCRIPT)
	@echo - Installer created successfully!

clean:
	@echo - Cleaning up...
	@del rsrc_windows_*.syso
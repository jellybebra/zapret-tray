package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/getlantern/systray"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const ServiceName = "zapret"

//go:embed icon.ico
var iconData []byte

type ServiceAction int

const (
	ActionStart ServiceAction = iota
	ActionStop
)

func main() {
	// 1. Проверяем права администратора
	if !isAdmin() {
		runMeElevated()
		return
	}

	// 2. Запускаем приложение в трее
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(iconData)
	systray.SetTitle("Zapret Controller")
	systray.SetTooltip("Управление Zapret")

	// Элементы меню
	mStatus := systray.AddMenuItem("Состояние: Проверка...", "Текущий статус службы")
	mStatus.Disable()
	systray.AddSeparator()
	mStart := systray.AddMenuItem("Запустить", "Start Service")
	mStop := systray.AddMenuItem("Остановить", "Stop Service")
	mRestart := systray.AddMenuItem("Перезагрузить", "Restart Service")
	systray.AddSeparator()

	// Versions Submenu
	mVersions := systray.AddMenuItem("Версии", "Управление версиями")
	mRefreshVersions := mVersions.AddSubMenuItem("Обновить список версий", "Обновить список версий")
	systray.AddSeparator() // Separator in main menu

	mOpenBat := systray.AddMenuItem("Открыть service.bat", "Открыть папку со скриптом")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выход", "Закрыть программу")

	// Dynamic items storage
	var versionItems []*systray.MenuItem

	// Declare ahead to allow recursion if needed (though we call it from a goroutine)
	var refreshVersionsList func()

	refreshVersionsList = func() {
		mVersions.SetTitle("Версии (Loading...)")
		go func() {
			versions, err := GetAllVersions()

			// Update UI on main thread (systray is thread safe mostly, but logic should be linear)
			// But we need to clear old items.
			// systray doesn't have Remove. We will Hide() old items.
			for _, item := range versionItems {
				item.Hide()
			}

			versionItems = nil // Abandon old ones (they stay in memory but hidden)

			if err != nil {
				mVersions.SetTitle("Версии (Error)")
				return
			}
			mVersions.SetTitle("Версии")

			for _, v := range versions {
				title := v.Name
				tooltip := ""
				if v.IsInstalled {
					// "скачанные официальные версии (disabled, пример: "zapret-1.9.3", отображаться должно только "1.9.3")"
					// "скачанные кастомные версии (disabled... отображать надо все название целиком)"
					// My `v.Name` already handles the text logic (trimmed or full).
					if v.IsCustom {
						title = v.Name + " (установлено)"
					} else {
						title = v.Name + " (установлено)"
					}
				} else {
					title = v.Name + " (скачать)"
				}

				item := mVersions.AddSubMenuItem(title, tooltip)
				versionItems = append(versionItems, item)

				if v.IsInstalled {
					item.Disable()
				} else {
					// Setup click handler for download
					vCopy := v // Capture loop var
					go func(itm *systray.MenuItem, ver Version) {
						for range itm.ClickedCh {
							log.Printf("Global download requested for %s", ver.Name)
							itm.SetTitle("Downloading... " + ver.Name)
							itm.Disable()

							err := DownloadVersion(ver)
							if err != nil {
								log.Println("Download failed:", err)
								itm.SetTitle("Error: " + ver.Name)
								itm.Enable()
							} else {
								log.Println("Download finished")
								itm.SetTitle(ver.Name + " (Installed)")
								// Trigger full refresh to update state properly
								refreshVersionsList()
							}
						}
					}(item, vCopy)
				}
			}
		}()
	}

	// Initial call
	refreshVersionsList()

	// Канал для обновления статуса
	go func() {
		for {
			state, err := getServiceStatus(ServiceName)
			if err != nil {
				mStatus.SetTitle(fmt.Sprintf("Ошибка: %v", err))
			} else {
				statusText := "Неизвестно"
				switch state {
				case svc.Stopped:
					statusText = "Остановлен"
				case svc.StartPending:
					statusText = "Запускается..."
				case svc.StopPending:
					statusText = "Останавливается..."
				case svc.Running:
					statusText = "Работает"
				}
				mStatus.SetTitle(fmt.Sprintf("Состояние: 🟢 %s", statusText))
				systray.SetTooltip(fmt.Sprintf("Zapret Controller: %s", statusText))

				// Управление активностью кнопок
				if state == svc.Running {
					mStart.Disable()
					mStop.Enable()
					mRestart.Enable()
				} else if state == svc.Stopped {
					mStart.Enable()
					mStop.Disable()
					mRestart.Disable()
				}
			}
			time.Sleep(2 * time.Second)
		}
	}()

	// Обработка нажатий
	go func() {
		for {
			select {
			case <-mRefreshVersions.ClickedCh:
				refreshVersionsList()
			case <-mStart.ClickedCh:
				controlService(ServiceName, ActionStart)
			case <-mStop.ClickedCh:
				controlService(ServiceName, ActionStop)
			case <-mRestart.ClickedCh:
				// Рестарт: Стоп -> Ждем -> Старт
				controlService(ServiceName, ActionStop)
				time.Sleep(1 * time.Second)
				controlService(ServiceName, ActionStart)

			case <-mOpenBat.ClickedCh:
				openServiceBat()
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()
}

func onExit() {
	// Очистка при выходе
}

// === ФУНКЦИИ РАБОТЫ С СЕРВИСОМ ===

func getServiceStatus(name string) (svc.State, error) {
	m, err := mgr.Connect()
	if err != nil {
		return 0, err
	}
	defer func(m *mgr.Mgr) {
		err := m.Disconnect()
		if err != nil {

		}
	}(m)

	s, err := m.OpenService(name)
	if err != nil {
		return 0, err
	}
	defer func(s *mgr.Service) {
		err := s.Close()
		if err != nil {

		}
	}(s)

	status, err := s.Query()
	if err != nil {
		return 0, err
	}
	return status.State, nil
}

// controlService теперь принимает нашу кастомную Action, а не svc.Cmd
func controlService(name string, action ServiceAction) {
	m, err := mgr.Connect()
	if err != nil {
		log.Println("SCM connection failed:", err)
		return
	}
	defer func(m *mgr.Mgr) {
		err := m.Disconnect()
		if err != nil {

		}
	}(m)

	s, err := m.OpenService(name)
	if err != nil {
		log.Println("Service open failed:", err)
		return
	}
	defer func(s *mgr.Service) {
		err := s.Close()
		if err != nil {

		}
	}(s)

	if action == ActionStart {
		err = s.Start()
	} else if action == ActionStop {
		_, err = s.Control(svc.Stop)
	}

	if err != nil {
		log.Println("Service control error:", err)
	}
}

func isAdmin() bool {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	isUserAnAdmin := shell32.NewProc("IsUserAnAdmin")

	ret, _, _ := isUserAnAdmin.Call()
	return ret != 0
}

func runMeElevated() {
	verb := "runas"
	exe, _ := os.Executable()
	cwd, _ := os.Getwd()
	args := strings.Join(os.Args[1:], " ")

	verbPtr, _ := windows.UTF16PtrFromString(verb)
	exePtr, _ := windows.UTF16PtrFromString(exe)
	cwdPtr, _ := windows.UTF16PtrFromString(cwd)
	argsPtr, _ := windows.UTF16PtrFromString(args)

	var showCmd int32 = 1 //SW_NORMAL

	err := windows.ShellExecute(0, verbPtr, exePtr, argsPtr, cwdPtr, showCmd)
	if err != nil {
		fmt.Println(err)
	}
}

func getServiceBinaryPath(name string) (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", err
	}
	defer func(m *mgr.Mgr) {
		err := m.Disconnect()
		if err != nil {

		}
	}(m)

	s, err := m.OpenService(name)
	if err != nil {
		return "", err
	}
	defer func(s *mgr.Service) {
		err := s.Close()
		if err != nil {

		}
	}(s)

	config, err := s.Config()
	if err != nil {
		return "", err
	}
	return config.BinaryPathName, nil
}

func openServiceBat() {
	rawPath, err := getServiceBinaryPath(ServiceName)
	if err != nil {
		log.Println("Не удалось получить путь к сервису:", err)
		return
	}

	// Очистка пути от кавычек и аргументов
	exePath := rawPath
	if len(exePath) > 0 && exePath[0] == '"' {
		// Путь в кавычках (например "C:\Path\To\exe")
		if end := strings.Index(exePath[1:], "\""); end != -1 {
			exePath = exePath[1 : end+1]
		}
	} else {
		// Путь без кавычек; берем до первого пробела, если есть аргументы
		parts := strings.Split(exePath, " ")
		if len(parts) > 0 {
			exePath = parts[0]
		}
	}

	// Определяем директорию
	dir := filepath.Dir(exePath)
	// Если мы внутри bin, поднимаемся на уровень выше
	if strings.ToLower(filepath.Base(dir)) == "bin" {
		dir = filepath.Dir(dir)
	}

	batPath := filepath.Join(dir, "service.bat")
	log.Println("Открываем:", batPath)

	// Запускаем через cmd start
	err = exec.Command("cmd", "/c", "start", "", batPath).Start()
	if err != nil {
		log.Println("Ошибка запуска service.bat:", err)
	}
}

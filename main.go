package main

import (
	_ "embed"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/getlantern/systray"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// === НАСТРОЙКИ ===
// Укажите здесь имя сервиса как в системе (например "zapret" или "winws")
const ServiceName = "zapret"

//go:embed icon.ico
var iconData []byte

// Определяем свои константы действий, так как svc.Start не существует
type ServiceAction int

const (
	ActionStart ServiceAction = iota
	ActionStop
)

func main() {
	// 1. Проверяем права администратора
	if !amAdmin() {
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
	mQuit := systray.AddMenuItem("Выход", "Закрыть программу")

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
					statusText = "🔴 Остановлен"
				case svc.StartPending:
					statusText = "🟡 Запускается..."
				case svc.StopPending:
					statusText = "🟡 Останавливается..."
				case svc.Running:
					statusText = "🟢 Работает"
				}
				mStatus.SetTitle(fmt.Sprintf("Состояние: %s", statusText))

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
			case <-mStart.ClickedCh:
				controlService(ServiceName, ActionStart)
			case <-mStop.ClickedCh:
				controlService(ServiceName, ActionStop)
			case <-mRestart.ClickedCh:
				// Рестарт: Стоп -> Ждем -> Старт
				controlService(ServiceName, ActionStop)
				time.Sleep(1 * time.Second)
				controlService(ServiceName, ActionStart)
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
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return 0, err
	}
	defer s.Close()

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
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		log.Println("Service open failed:", err)
		return
	}
	defer s.Close()

	if action == ActionStart {
		// Для старта вызывается метод .Start(), а не .Control()
		err = s.Start()
	} else if action == ActionStop {
		// Для стопа отправляется сигнал svc.Stop
		_, err = s.Control(svc.Stop)
	}

	if err != nil {
		log.Println("Service control error:", err)
	}
}

// === ФУНКЦИИ ДЛЯ ПРАВ АДМИНИСТРАТОРА ===

func amAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
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

# 🛡️ Zapret Tray


**Zapret Tray** — это удобная обертка над [zapret-discord-youtube](https://github.com/Flowseal/zapret-discord-youtube), которая позволяет управлять версиями и службой `zapret`.

Больше не нужно лазить в `services.msc` или запускать скрипты вручную.

<img width="567" height="652" alt="image" src="https://github.com/user-attachments/assets/2ca96302-bda1-4366-b8fd-f9aa7e75f717" />

## ✨ Возможности

*   **⬇️ Поддержка кастомных версий:** Просто положите их в папку рядом с другими версиями
*   **🧹 Чистое удаление:** Деинсталлятор корректно удаляет службу, драйвер `WinDivert` и все хвосты

## 📥 Установка

1.  Скачайте и запустите актуальный установщик `ZapretTraySetup.exe` из раздела [Releases](#)
2.  После запуска в трее (возле часов) появится иконка
3.  Нажмите правой кнопкой мыши по иконке -> **Версии**
4.  Выберите актуальную версию и нажмите, чтобы скачать её
5.  Нажмите на установленную версию, чтобы открыть её service.bat
6.  Пользуйтесь как обычно

## 🛠️ Сборка из исходного кода

Если вы хотите скомпилировать приложение самостоятельно или внести изменения.

### Требования

*   **Go** 1.20+
*   **Make** (опционально, для использования Makefile)
*   **Inno Setup 6** (для создания установщика)
*   [go-winres](https://github.com/tc-hib/go-winres) (для генерации ресурсов Windows/иконок)

### Инструкция по сборке

1.  **Клонируйте репозиторий:**
    ```bash
    git clone https://github.com/jellybebra/zapret-tray.git
    cd zapret-tray
    ```

2.  **Установите генератор ресурсов:**
    ```bash
    go install github.com/tc-hib/go-winres@latest
    ```

3.  **Сборка EXE (через Make):**
    Команда сгенерирует ресурсы и соберет бинарный файл `zapret-tray.exe`.
    ```bash
    make release
    ```
    *Без Make:*
    ```bash
    go-winres make --arch amd64
    go build -o zapret-tray.exe -ldflags="-s -w -H windowsgui"
    ```

4.  **Создание установщика:**
    Убедитесь, что Inno Setup установлен. Путь к компилятору указан в `Makefile`.
    ```bash
    make installer
    ```
    На выходе вы получите файл `Output/ZapretTraySetup.exe`.

## ⚖️ Legal & Credits

This application is a GUI wrapper and downloader. It relies on the following projects:

1.  **Zapret** by [bol-van](https://github.com/bol-van/zapret) (MIT License).
2.  **Zapret-discord-youtube** builds by [Flowseal](https://github.com/Flowseal/zapret-discord-youtube) (MIT License).
3.  **WinDivert** by [basil00](https://github.com/basil00/WinDivert) (LGPLv3 / GPLv2).

The binaries downloaded by this application utilize WinDivert. 
The source code for WinDivert is available at https://github.com/basil00/WinDivert.
This application does not modify WinDivert binaries.

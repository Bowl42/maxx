package desktop

import (
	"log"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
)

type TrayManager struct {
	app      *DesktopApp
	trayMenu *menu.TrayMenu
}

func NewTrayManager(app *DesktopApp) *TrayManager {
	tm := &TrayManager{
		app: app,
	}

	tm.buildMenu()

	log.Println("[Tray] System tray initialized")
	return tm
}

func (tm *TrayManager) buildMenu() {
	appMenu := menu.NewMenu()
	
	appMenu.Append(menu.Text("打开管理面板", keys.CmdOrCtrl("M"), func(_ *menu.CallbackData) {
		tm.app.ShowWindow()
	}))
	appMenu.Append(menu.Separator())
	appMenu.Append(menu.Text("🌐 服务器地址", nil, func(_ *menu.CallbackData) {
		tm.app.CopyServerAddress()
	}))
	appMenu.Append(menu.Text("📋 复制服务器地址", keys.CmdOrCtrl("C"), func(_ *menu.CallbackData) {
		tm.app.CopyServerAddress()
	}))
	appMenu.Append(menu.Separator())
	appMenu.Append(menu.Checkbox("启动服务器", true, nil, func(_ *menu.CallbackData) {
		if err := tm.app.StartServer(); err != nil {
			log.Printf("[Tray] Failed to start server: %v", err)
		}
	}))
	appMenu.Append(menu.Text("🔄 重启服务器", nil, func(_ *menu.CallbackData) {
		if err := tm.app.RestartServer(); err != nil {
			log.Printf("[Tray] Failed to restart server: %v", err)
		}
	}))
	appMenu.Append(menu.Text("⏸️ 停止服务器", nil, func(_ *menu.CallbackData) {
		if err := tm.app.StopServer(); err != nil {
			log.Printf("[Tray] Failed to stop server: %v", err)
		}
	}))
	appMenu.Append(menu.Separator())
	appMenu.Append(menu.Checkbox("开机自启动", tm.app.IsAutoStartEnabled(), nil, func(cd *menu.CallbackData) {
		enabled := cd.MenuItem.Checked
		if err := tm.app.SetAutoStart(enabled); err != nil {
			log.Printf("[Tray] Failed to set auto-start: %v", err)
		}
	}))
	appMenu.Append(menu.Checkbox("托盘模式", tm.app.IsTrayMode(), nil, func(cd *menu.CallbackData) {
		enabled := cd.MenuItem.Checked
		if err := tm.app.SetTrayMode(enabled); err != nil {
			log.Printf("[Tray] Failed to set tray mode: %v", err)
		}
	}))
	appMenu.Append(menu.Separator())
	appMenu.Append(menu.Text("📁 打开数据目录", nil, func(_ *menu.CallbackData) {
		if err := tm.app.OpenDataDir(); err != nil {
			log.Printf("[Tray] Failed to open data directory: %v", err)
		}
	}))
	appMenu.Append(menu.Text("📜 查看日志", nil, func(_ *menu.CallbackData) {
		if err := tm.app.OpenLogFile(); err != nil {
			log.Printf("[Tray] Failed to open log file: %v", err)
		}
	}))
	appMenu.Append(menu.Separator())
	appMenu.Append(menu.Text("退出", keys.CmdOrCtrl("Q"), func(_ *menu.CallbackData) {
		tm.app.Quit()
	}))

	tm.trayMenu = &menu.TrayMenu{
		Label: "maxx",
		Menu:  appMenu,
	}
	
	// System tray will be shown automatically by Wails
	// No need to call runtime.MenuSetTrayMenu
}

func (tm *TrayManager) IsTrayMode() bool {
	return tm.app.IsTrayMode()
}

func (tm *TrayManager) UpdateServerStatus(status string) {
	log.Printf("[Tray] Updating server status: %s", status)
	tm.buildMenu()
}

func (tm *TrayManager) Shutdown() {
	log.Println("[Tray] Shutting down tray manager")
}

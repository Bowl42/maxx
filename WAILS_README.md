# maxx-next Desktop Application (Wails)

## Overview

maxx-next 现在支持两种运行模式：

1. **服务器模式** (Server Mode): Docker 部署，浏览器远程访问
2. **桌面模式** (Desktop Mode): Wails 桌面应用，Windows 优先支持

## Prerequisites

- Go 1.21+
- Node.js 18+
- Wails CLI
- Windows (for desktop app)

## Installation

### Install Wails CLI
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Install Dependencies
```bash
task install
# or
task install:frontend
task install:wails
```

## Development

### Server Mode (传统模式)
```bash
# Backend
task dev:backend

# Frontend
task dev:frontend

# Both
task dev
```

访问: http://localhost:9880

### Desktop Mode (Wails 应用)
```bash
task dev:desktop
# or
wails dev
```

## Building

### Build Server Binary
```bash
task build
# or
task build:backend
```

### Build Desktop Application
```bash
# Windows
task build:desktop:windows

# Or run the build script
build-desktop.bat

# Cross-platform (using Wails)
wails build -platform windows/amd64
wails build -platform darwin/amd64
wails build -platform linux/amd64
```

Output location:
- Windows: `build/bin/maxx.exe`
- macOS: `build/bin/maxx.app`
- Linux: `build/bin/maxx`

## Features

### Desktop Mode Features
- ✅ System Tray Integration (系统托盘集成)
- ✅ Auto-start on boot (开机自启)
- ✅ Minimize to tray (最小化到托盘)
- ✅ Embedded HTTP Server (内嵌 HTTP 服务器)
- ✅ Native file system access (原生文件访问)
- ✅ System notifications (系统通知)

### Server Mode Features
- ✅ Docker container support
- ✅ Remote browser access
- ✅ All existing features (API, WebSocket, etc.)

## Architecture

```
maxx-next/
├── cmd/
│   ├── maxx/main.go          # Server mode entry
│   └── desktop/main.go       # Wails desktop entry
├── internal/
│   ├── core/                 # Shared core (server + database)
│   ├── desktop/              # Desktop app logic
│   │   ├── app.go          # Desktop app core
│   │   ├── api.go          # Wails bindings (AdminService proxy)
│   │   ├── tray.go         # System tray manager
│   │   └── autostart.go    # Windows auto-start
│   └── [existing modules]
├── web/                      # React frontend (shared)
├── wails.json               # Wails configuration
├── build-desktop.bat         # Windows build script
└── Dockerfile               # Server mode Dockerfile
```

## Data Directory

### Server Mode
- Windows: `%APPDATA%\maxx` or `~/.config/maxx`
- Docker: `/data`

### Desktop Mode
- Windows: `%APPDATA%\maxx`
- macOS: `~/Library/Application Support/maxx`
- Linux: `~/.config/maxx`

## Usage

### Desktop App

1. **Launch**: Run `maxx.exe` (Windows)
2. **Tray Menu**: Right-click system tray icon
   - Open Dashboard
   - Copy Server Address
   - Start/Stop/Restart Server
   - Toggle Auto-start
   - Open Data Directory
   - View Logs
   - Quit
3. **Window Behavior**: Clicking X hides to tray (tray mode)
4. **Show Window**: Double-click tray icon or use hotkey

### System Tray Features

- 🟢 Running indicator
- 📊 Open management panel
- 🌐 Copy server address
- 🔄 Restart service
- 📁 Open data directory
- 📜 View log file
- ⏸️ Stop service
- 🔇/🔊 Toggle auto-start
- ❌ Quit application

## API Compatibility

Both modes support the same Admin API:

- Provider CRUD
- Project CRUD
- Route CRUD
- Session Management
- Retry Config CRUD
- Routing Strategy CRUD
- Proxy Request Query
- Statistics
- Settings

### Desktop Mode API Calls

In desktop mode, the frontend uses Wails bindings instead of HTTP:

```typescript
// Automatically detected by runtime
const isWails = window.__WAILS__;

// API calls go through Wails runtime
const providers = await window.wails.Call('DesktopApp.GetProviders');
```

In server mode:
```typescript
// API calls go through HTTP
const providers = await fetch('/admin/providers').then(r => r.json());
```

## Troubleshooting

### Build Issues

**"wails: command not found"**
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
# Add Go bin to PATH
```

**"frontend not built"**
```bash
cd web
npm run build
cd ..
```

**"module not found"**
```bash
go mod tidy
go mod download
```

### Runtime Issues

**Server won't start**
- Check port 9880 is not in use
- Check data directory permissions
- View logs in data directory

**Tray not showing**
- Restart the application
- Check Windows notifications are enabled

**Window won't show**
- Check if app is running in background
- Use tray menu -> "Open Dashboard"

## Development Tips

### Hot Reload (Desktop Mode)
```bash
# Terminal 1: Wails dev server
wails dev

# Terminal 2: Frontend dev server (optional, if needed)
cd web
npm run dev
```

### Debugging Wails Bindings
- Check `wailsjs/runtime` generated files
- Use `window.wails.Call()` in browser console
- Enable verbose logging in `app.go`

### Testing API Methods
```go
// Test in Go (desktop mode)
func (a *DesktopApp) TestMethod() string {
    return "Hello from Wails!"
}
```

```javascript
// Call from frontend
const result = await window.wails.Call('DesktopApp.TestMethod');
console.log(result); // "Hello from Wails!"
```

## Security Considerations

- Desktop mode runs HTTP server on localhost:9880
- Port is accessible only to local machine
- Use firewall rules to restrict access if needed
- Data directory contains sensitive API keys - protect appropriately

## Performance

- Startup time: < 3 seconds
- Memory usage: ~50-100 MB (idle)
- Disk usage: SQLite database + logs (~10-50 MB)

## Future Enhancements

- [ ] macOS and Linux desktop support
- [ ] Native notifications integration
- [ ] Auto-updater
- [ ] Crash reporting
- [ ] Telemetry (opt-in)
- [ ] Custom hotkey configuration
- [ ] System tray icon customization
- [ ] Dark mode system integration

## License

Same as maxx-next project.

## Support

- GitHub Issues: https://github.com/Bowl42/maxx-next/issues
- Documentation: [Link to docs]
- Wails Docs: https://wails.io/docs

# Quick Start Guide - maxx-next Desktop App

## 🚀 First Time Setup (5 minutes)

### 1. Install Wails CLI (1 min)
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### 2. Install Dependencies (2 mins)
```bash
task install
# or
cd web && npm install
```

### 3. Build Frontend (1 min)
```bash
task build:frontend
# or
cd web && npm run build
```

### 4. Run Desktop App (1 min)
```bash
wails dev
```

Or use the build script (Windows):
```bash
build-desktop.bat
```

## ✅ Verification

Check these items to verify everything works:

- [ ] Wails app starts successfully
- [ ] Window opens with frontend loaded
- [ ] System tray icon appears
- [ ] HTTP server auto-starts on port 9880
- [ ] "Server Status" shows "running"
- [ ] Can open admin panel from tray menu
- [ ] Can copy server address to clipboard
- [ ] Can open data directory
- [ ] Window minimizes to tray when closed
- [ ] App quits properly from tray menu

## 🎯 Common Tasks

### Start Development Server
```bash
# Desktop mode
wails dev

# Server mode (browser)
task dev
```

### Build Release
```bash
# Windows (use build script)
build-desktop.bat

# Cross-platform
wails build -platform windows/amd64
wails build -platform darwin/amd64
wails build -platform linux/amd64
```

### View Logs
Desktop mode logs are at:
- Windows: `%APPDATA%\maxx\maxx.log`
- Or use tray menu: "📜 查看日志"

### Reset Everything
```bash
task clean
task install
task build:frontend
```

## 🐛 Troubleshooting

### "wails: command not found"
```bash
# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest
# Add Go bin to PATH
```

### "frontend not built"
```bash
cd web
npm run build
cd ..
```

### "can't find package github.com/wailsapp/wails/v2"
```bash
go mod tidy
go mod download
```

### Window doesn't open
- Check if app is running in Task Manager
- Kill existing `maxx.exe` processes
- Try running again

### System tray not showing
- Check Windows notification settings
- Restart the application
- Check if app is running (Task Manager)

### Port 9880 already in use
```bash
# Find process using port
netstat -ano | findstr :9880

# Kill the process or change port in internal/desktop/app.go
# Look for: serverPort = ":9880"
```

## 📁 What Gets Created

When you build the desktop app:
```
build/
├── bin/
│   └── maxx.exe              # Main executable (6-15 MB)
├── icon.ico                  # Icon files
└── wails.json                # Build metadata
```

## 🔧 Configuration

### Change Default Port
Edit `internal/desktop/app.go`:
```go
serverPort = ":9881"  // Change this line
```

### Change Window Size
Edit `cmd/desktop/main.go`:
```go
options.App{
    Width:  1400,  // Change this
    Height: 900,
    // ...
}
```

### Disable Tray Mode
Edit `internal/desktop/app.go`:
```go
trayMode: false,  // Change this line
```

## 📞 Getting Help

1. Check `WAILS_README.md` for detailed documentation
2. Check `IMPLEMENTATION_SUMMARY.md` for implementation details
3. Review console output for errors
4. Check log files in data directory
5. Open GitHub issue with error details

## 🎓 Next Steps After Setup

1. Test all features ( Providers, Routes, Projects, etc.)
2. Configure providers and routes
3. Test AI API endpoints
4. Monitor requests in real-time
5. Set up auto-start if desired
6. Customize tray settings

## 🎉 You're Ready!

 maxx-next desktop application is now ready to use. Enjoy!

---

**Need more help?** See full documentation in `WAILS_README.md`

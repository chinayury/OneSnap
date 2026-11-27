package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/image/bmp"
	"golang.org/x/image/webp"
	"onesnap/internal/config"
	"onesnap/internal/hotkeys"
	"onesnap/internal/library"
	"onesnap/internal/overlay"
	"onesnap/internal/screenshot"
	"onesnap/internal/tray"
	"onesnap/internal/updater"
	winEnum "onesnap/internal/windows"
)

// Version is set at build time via ldflags
var Version = "dev"

// App struct
type App struct {
	ctx              context.Context
	hotkeyManager    *hotkeys.HotkeyManager
	overlayManager   *overlay.Manager
	trayIcon         *tray.TrayIcon
	config           *config.Config
	lastWidth        int
	lastHeight       int
	preCaptureWidth  int // Window size before capture (protected from resize events)
	preCaptureHeight int
	preCaptureX      int  // Window X position before capture
	preCaptureY      int  // Window Y position before capture
	isCapturing      bool // Flag to prevent resize events during capture
	isWindowHidden   bool // Track window visibility state
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}
	a.config = cfg

	// Initialize hotkey manager
	a.hotkeyManager = hotkeys.NewHotkeyManager()
	a.hotkeyManager.SetCallback(a.onHotkey)

	// Register hotkeys from config
	a.registerHotkeysFromConfig()
	a.hotkeyManager.Start()

	// Initialize overlay manager for native region selection
	a.overlayManager = overlay.NewManager()
	if err := a.overlayManager.Start(); err != nil {
		// Log warning but continue - will fall back to React overlay
		println("Warning: failed to start overlay manager:", err.Error())
	}

	// Initialize system tray with version in tooltip
	a.trayIcon = tray.NewTrayIcon("OneSnap " + Version)
	a.trayIcon.SetCallback(a.onTrayMenu)
	a.trayIcon.SetLanguage(a.config.Language)
	a.trayIcon.SetOnShow(func() {
		runtime.WindowShow(a.ctx)
		a.isWindowHidden = false
		runtime.WindowSetAlwaysOnTop(a.ctx, true)
		runtime.WindowSetAlwaysOnTop(a.ctx, false)
	})
	a.trayIcon.Start()

	// Initialize window size tracking with config values
	a.lastWidth = cfg.Window.Width
	a.lastHeight = cfg.Window.Height

	// Track hidden state if app started minimized to tray (set via Wails StartHidden option)
	if cfg.Startup.MinimizeToTray {
		a.isWindowHidden = true
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	// Save window size before closing using tracked values
	if a.config != nil && a.lastWidth >= 800 && a.lastHeight >= 600 {
		a.config.Window.Width = a.lastWidth
		a.config.Window.Height = a.lastHeight
		a.config.Save()
	}

	// Cleanup resources
	if a.hotkeyManager != nil {
		a.hotkeyManager.Stop()
		a.hotkeyManager.UnregisterAll()
	}
	if a.overlayManager != nil {
		a.overlayManager.Stop()
	}
	if a.trayIcon != nil {
		a.trayIcon.Stop()
	}
}

// onHotkey handles global hotkey events
func (a *App) onHotkey(id int) {
	switch id {
	case hotkeys.HotkeyFullscreen:
		// Quick-save mode: capture and save silently, no editor window
		if a.config.Startup.CaptureMode == "quicksave" {
			go a.quickSaveFullscreen()
			return
		}
		runtime.EventsEmit(a.ctx, "hotkey:fullscreen")
	case hotkeys.HotkeyRegion:
		runtime.EventsEmit(a.ctx, "hotkey:region")
	case hotkeys.HotkeyWindow:
		// Quick-save mode: capture the foreground window silently
		if a.config.Startup.CaptureMode == "quicksave" {
			go a.quickSaveForegroundWindow()
			return
		}
		runtime.EventsEmit(a.ctx, "hotkey:window")
	}
}

// quickSaveFullscreen captures the display under the cursor and saves it
// without opening the editor window (quick-save mode).
func (a *App) quickSaveFullscreen() {
	result, err := screenshot.CaptureFullscreen()
	if err != nil {
		a.notifyQuickSaveFailure(err)
		return
	}
	data, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		a.notifyQuickSaveFailure(err)
		return
	}
	a.saveAndNotify(data, ".png")
}

// quickSaveForegroundWindow captures the current foreground window and
// saves it without opening the editor window (quick-save mode).
func (a *App) quickSaveForegroundWindow() {
	result, err := screenshot.CaptureWindowByCoords(screenshot.GetForegroundWindow())
	if err != nil {
		a.notifyQuickSaveFailure(err)
		return
	}
	data, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		a.notifyQuickSaveFailure(err)
		return
	}
	a.saveAndNotify(data, ".png")
}

// saveAndNotify writes image bytes to the quick-save folder and shows a
// localized tray balloon with the saved path (or the failure reason).
func (a *App) saveAndNotify(data []byte, ext string) {
	filePath, err := a.quickSaveFile(data, ext)
	if a.trayIcon == nil {
		return
	}
	if err != nil {
		a.trayIcon.ShowBalloon("OneSnap", a.quickSaveFailText(err))
		return
	}
	a.trayIcon.ShowBalloon("OneSnap", a.quickSaveSavedText(filePath))
}

func (a *App) notifyQuickSaveFailure(err error) {
	if a.trayIcon != nil {
		a.trayIcon.ShowBalloon("OneSnap", a.quickSaveFailText(err))
	}
}

// quickSaveSavedText / quickSaveFailText return localized balloon texts.
func (a *App) quickSaveSavedText(path string) string {
	if a.config.Language == "en" {
		return "Saved: " + path
	}
	return "已保存：" + path
}

func (a *App) quickSaveFailText(err error) string {
	if a.config.Language == "en" {
		return "Quick save failed: " + err.Error()
	}
	return "快速保存失败：" + err.Error()
}

// quickSaveFile writes image bytes to the configured quick-save folder using
// the configured naming pattern. Returns the full file path.
func (a *App) quickSaveFile(data []byte, ext string) (string, error) {
	saveDir := a.config.QuickSave.Folder
	if saveDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		saveDir = filepath.Join(homeDir, "Pictures", "OneSnap")
	}

	// Create save directory if it doesn't exist
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create save directory: %w", err)
	}

	now := time.Now()
	pattern := a.config.QuickSave.Pattern
	stem := "onesnap_" + now.Format("2006-01-02_15-04-05")
	if pattern == "date" {
		stem = "onesnap_" + now.Format("2006-01-02")
	}

	for index := 0; ; index++ {
		name := stem
		if pattern == "increment" {
			name = fmt.Sprintf("onesnap_%03d", index+1)
		} else if index > 0 {
			name = fmt.Sprintf("%s_%d", stem, index)
		}
		filePath := filepath.Join(saveDir, name+ext)
		file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("failed to create screenshot: %w", err)
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			os.Remove(filePath)
			return "", fmt.Errorf("failed to save file: %w", err)
		}
		return filePath, nil
	}
}

// onTrayMenu handles tray menu selections
func (a *App) onTrayMenu(menuID int) {
	switch menuID {
	case tray.MenuFullscreen:
		runtime.EventsEmit(a.ctx, "hotkey:fullscreen")
	case tray.MenuRegion:
		runtime.EventsEmit(a.ctx, "hotkey:region")
	case tray.MenuWindow:
		runtime.EventsEmit(a.ctx, "hotkey:window")
	case tray.MenuLibrary:
		// Show main window first so library modal has context
		runtime.WindowShow(a.ctx)
		a.isWindowHidden = false
		// Emit event to open library window
		runtime.EventsEmit(a.ctx, "tray:library")
	case tray.MenuLanguage:
		newLang := "en"
		if a.config.Language == "en" {
			newLang = "zh"
		}
		a.config.Language = newLang
		_ = a.config.Save()
		if a.trayIcon != nil {
			a.trayIcon.SetLanguage(newLang)
		}
		runtime.EventsEmit(a.ctx, "lang:changed", newLang)
	case tray.MenuQuit:
		// Quit the application - use goroutine to avoid blocking tray menu
		go func() {
			// First try graceful shutdown via runtime.Quit
			// This may not work reliably on Windows from a goroutine
			runtime.Quit(a.ctx)

			// Fallback: force exit after a short delay if Quit didn't work
			time.Sleep(500 * time.Millisecond)
			os.Exit(0)
		}()
	}
}

// UpdateWindowSize tracks the current window size for persistence
func (a *App) UpdateWindowSize(width, height int) {
	// Skip updates during capture to preserve pre-capture size
	if a.isCapturing {
		return
	}
	if width >= 800 && height >= 600 {
		a.lastWidth = width
		a.lastHeight = height
	}
}

// MinimizeToTray hides the window (minimize to tray)
func (a *App) MinimizeToTray() {
	runtime.WindowHide(a.ctx)
	a.isWindowHidden = true
}

// OnBeforeClose is called when the window close button is clicked
// Returns true to prevent the default close behavior (if close-to-tray is enabled)
func (a *App) OnBeforeClose(ctx context.Context) bool {
	if a.config != nil && a.config.Startup.CloseToTray {
		// Hide window instead of closing
		runtime.WindowHide(ctx)
		a.isWindowHidden = true
		return true // Prevent default close
	}
	return false // Allow normal close
}

// VirtualScreenBounds represents the combined bounds of all monitors
type VirtualScreenBounds struct {
	X      int `json:"x"` // Can be negative (monitor left of primary)
	Y      int `json:"y"` // Can be negative (monitor above primary)
	Width  int `json:"width"`
	Height int `json:"height"`
}

// RegionCaptureData holds the fullscreen screenshot and display info for region selection
type RegionCaptureData struct {
	Screenshot   *screenshot.CaptureResult `json:"screenshot"`
	ScreenX      int                       `json:"screenX"`
	ScreenY      int                       `json:"screenY"`
	Width        int                       `json:"width"`
	Height       int                       `json:"height"`
	ScaleRatio   float64                   `json:"scaleRatio"`   // DPI scale ratio (physical/logical)
	PhysicalW    int                       `json:"physicalW"`    // Actual screenshot width
	PhysicalH    int                       `json:"physicalH"`    // Actual screenshot height
	DisplayIndex int                       `json:"displayIndex"` // Index of the captured display
}

// PrepareRegionCapture prepares for region selection using native Win32 overlay
func (a *App) PrepareRegionCapture() (*RegionCaptureData, error) {
	// Set capturing flag to prevent resize events from overwriting saved size
	a.isCapturing = true

	// Save current window position before hiding
	a.preCaptureX, a.preCaptureY = runtime.WindowGetPosition(a.ctx)

	// Save current window size before hiding (protected from resize events)
	width, height := runtime.WindowGetSize(a.ctx)
	if width >= 800 && height >= 600 {
		a.preCaptureWidth = width
		a.preCaptureHeight = height
	} else if a.lastWidth >= 800 && a.lastHeight >= 600 {
		// Fallback to last tracked size
		a.preCaptureWidth = a.lastWidth
		a.preCaptureHeight = a.lastHeight
	} else {
		// Default fallback
		a.preCaptureWidth = 1200
		a.preCaptureHeight = 800
	}

	// Only hide and wait if window is currently visible
	wasVisible := !a.isWindowHidden
	quickSave := a.config.Startup.CaptureMode == "quicksave"
	if !a.isWindowHidden {
		runtime.WindowHide(a.ctx)
		a.isWindowHidden = true
		// Wait for window to fully hide (250ms for DWM compositor)
		time.Sleep(250 * time.Millisecond)
	}

	// Get the virtual screen bounds first
	screenX, screenY, virtualWidth, virtualHeight := screenshot.GetVirtualScreenBounds()

	// Capture raw RGBA (faster - no PNG encode)
	rgbaImg, err := screenshot.CaptureVirtualScreenRaw()
	if err != nil {
		runtime.WindowShow(a.ctx)
		a.isCapturing = false
		return nil, err
	}

	// Calculate scale ratio between physical screenshot and logical window size
	scaleRatio := float64(rgbaImg.Bounds().Dx()) / float64(virtualWidth)
	if scaleRatio < 1.0 {
		scaleRatio = 1.0
	}

	// Show native overlay and get result channel
	bounds := image.Rect(screenX, screenY, screenX+virtualWidth, screenY+virtualHeight)
	resultCh := a.overlayManager.Show(rgbaImg, bounds, scaleRatio, a.config.Language)

	// Wait for selection result in goroutine
	go func() {
		selResult := <-resultCh
		if selResult.Cancelled {
			// User cancelled - restore window only if it was visible before capture
			if wasVisible {
				runtime.WindowShow(a.ctx)
				a.isWindowHidden = false
			}
			a.isCapturing = false
			return
		}

		// Scale coordinates to physical pixels
		scaledX := int(float64(selResult.X) * scaleRatio)
		scaledY := int(float64(selResult.Y) * scaleRatio)
		scaledW := int(float64(selResult.Width) * scaleRatio)
		scaledH := int(float64(selResult.Height) * scaleRatio)

		// Crop to selected region before encoding (much faster - smaller image)
		croppedImg := rgbaImg.SubImage(image.Rect(scaledX, scaledY, scaledX+scaledW, scaledY+scaledH))

		var buf bytes.Buffer
		if err := png.Encode(&buf, croppedImg); err != nil {
			if wasVisible {
				runtime.WindowShow(a.ctx)
				a.isWindowHidden = false
			}
			a.isCapturing = false
			return
		}

		if quickSave {
			// Quick-save mode: save silently with a tray balloon, no editor window
			a.saveAndNotify(buf.Bytes(), ".png")
			// Restore the window only if it was visible before the capture
			if wasVisible {
				runtime.WindowShow(a.ctx)
				a.isWindowHidden = false
			}
			a.isCapturing = false
			return
		}

		// Emit cropped image directly - no need for frontend to crop again
		runtime.EventsEmit(a.ctx, "region:selected", map[string]interface{}{
			"width":      scaledW,
			"height":     scaledH,
			"screenshot": base64.StdEncoding.EncodeToString(buf.Bytes()),
		})
	}()

	// Return minimal data (actual selection comes via event)
	return &RegionCaptureData{
		Screenshot:   nil, // Not needed - selection via event
		ScreenX:      screenX,
		ScreenY:      screenY,
		Width:        virtualWidth,
		Height:       virtualHeight,
		ScaleRatio:   scaleRatio,
		PhysicalW:    rgbaImg.Bounds().Dx(),
		PhysicalH:    rgbaImg.Bounds().Dy(),
		DisplayIndex: 0,
	}, nil
}

// imageToRGBA converts an image.Image to *image.RGBA
func imageToRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return rgba
}

// FinishRegionCapture restores the window to normal state after region selection
func (a *App) FinishRegionCapture() {
	// Simply show window and clear capturing flag
	// Native overlay doesn't modify main window, so just show it
	runtime.WindowShow(a.ctx)
	a.isWindowHidden = false
	a.isCapturing = false
}

// ShowWindow shows the main window
func (a *App) ShowWindow() {
	runtime.WindowShow(a.ctx)
	a.isWindowHidden = false
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}

// CaptureFullscreen captures the display where the cursor is currently located
func (a *App) CaptureFullscreen() (*screenshot.CaptureResult, error) {
	return screenshot.CaptureFullscreen()
}

// CaptureRegion captures a specific region of the screen
func (a *App) CaptureRegion(x, y, width, height int) (*screenshot.CaptureResult, error) {
	return screenshot.CaptureRegion(x, y, width, height)
}

// CaptureDisplay captures a specific display by index
func (a *App) CaptureDisplay(displayIndex int) (*screenshot.CaptureResult, error) {
	return screenshot.CaptureDisplay(displayIndex)
}

// CaptureWindow captures a specific window by handle
func (a *App) CaptureWindow(hwnd int) (*screenshot.CaptureResult, error) {
	result, err := screenshot.CaptureWindowByCoords(uintptr(hwnd))

	// Bring OneSnap back to front after capture
	runtime.WindowShow(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	time.Sleep(50 * time.Millisecond)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)

	return result, err
}

// GetDisplayCount returns the number of active displays
func (a *App) GetDisplayCount() int {
	return screenshot.GetDisplayCount()
}

// GetActiveDisplayIndex returns the index of the display where the cursor is located
func (a *App) GetActiveDisplayIndex() int {
	return screenshot.GetMonitorAtCursor()
}

// GetVirtualScreenBounds returns the combined bounds of all monitors (virtual desktop)
func (a *App) GetVirtualScreenBounds() VirtualScreenBounds {
	x, y, w, h := screenshot.GetVirtualScreenBounds()
	return VirtualScreenBounds{X: x, Y: y, Width: w, Height: h}
}

// DisplayBounds represents the bounds of a display
type DisplayBounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// GetDisplayBounds returns the bounds of a display
func (a *App) GetDisplayBounds(displayIndex int) DisplayBounds {
	bounds := screenshot.GetDisplayBounds(displayIndex)
	return DisplayBounds{
		X:      bounds.Min.X,
		Y:      bounds.Min.Y,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
	}
}

// GetWindowList returns a list of all visible windows
func (a *App) GetWindowList() ([]winEnum.WindowInfo, error) {
	return winEnum.EnumWindows()
}

// GetWindowListWithThumbnails returns a list of all visible windows with thumbnails
func (a *App) GetWindowListWithThumbnails() ([]winEnum.WindowInfoWithThumbnail, error) {
	// Use 160x120 for thumbnails (4:3 aspect, good balance of quality and speed)
	return winEnum.EnumWindowsWithThumbnails(160, 120)
}

// GetWindowInfo returns information about a specific window
func (a *App) GetWindowInfo(hwnd int) (*winEnum.WindowInfo, error) {
	return winEnum.GetWindowInfo(uintptr(hwnd))
}

// SaveImageResult represents the result of saving an image
type SaveImageResult struct {
	Success  bool   `json:"success"`
	FilePath string `json:"filePath"`
	Error    string `json:"error,omitempty"`
}

// SaveImage saves a base64 encoded image to a file using a save dialog
func (a *App) SaveImage(imageData string, format string) SaveImageResult {
	// Determine file filter based on format
	var filters []runtime.FileFilter
	var defaultExt string

	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		filters = []runtime.FileFilter{{DisplayName: "JPEG Image", Pattern: "*.jpg;*.jpeg"}}
		defaultExt = ".jpg"
	default:
		filters = []runtime.FileFilter{{DisplayName: "PNG Image", Pattern: "*.png"}}
		defaultExt = ".png"
	}

	// Show save dialog
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Screenshot",
		DefaultFilename: "screenshot" + defaultExt,
		Filters:         filters,
	})

	if err != nil {
		return SaveImageResult{Success: false, Error: err.Error()}
	}

	if filePath == "" {
		return SaveImageResult{Success: false, Error: "No file selected"}
	}

	// Ensure correct extension
	ext := filepath.Ext(filePath)
	if ext == "" {
		filePath += defaultExt
	}

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(imageData)
	if err != nil {
		return SaveImageResult{Success: false, Error: "Failed to decode image data: " + err.Error()}
	}

	// Write to file
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return SaveImageResult{Success: false, Error: "Failed to save file: " + err.Error()}
	}

	return SaveImageResult{Success: true, FilePath: filePath}
}

// QuickSave saves a base64 encoded image to the configured directory
func (a *App) QuickSave(imageData string, format string) SaveImageResult {
	// Determine file extension
	ext := ".png"
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		ext = ".jpg"
	}

	// Decode base64 data
	data, err := base64.StdEncoding.DecodeString(imageData)
	if err != nil {
		return SaveImageResult{Success: false, Error: "Failed to decode image data: " + err.Error()}
	}

	filePath, err := a.quickSaveFile(data, ext)
	if err != nil {
		return SaveImageResult{Success: false, Error: err.Error()}
	}

	return SaveImageResult{Success: true, FilePath: filePath}
}

// HotkeyConfig represents a hotkey configuration
type HotkeyConfig struct {
	Fullscreen string `json:"fullscreen"`
	Region     string `json:"region"`
	Window     string `json:"window"`
}

// GetHotkeyConfig returns the current hotkey configuration
func (a *App) GetHotkeyConfig() HotkeyConfig {
	return HotkeyConfig{
		Fullscreen: a.config.Hotkeys.Fullscreen,
		Region:     a.config.Hotkeys.Region,
		Window:     a.config.Hotkeys.Window,
	}
}

// GetConfig returns the current application configuration
func (a *App) GetConfig() *config.Config {
	return a.config
}

// SaveConfig saves the application configuration
// The settings modal only manages hotkeys/startup/quicksave/export/update-check
// groups; everything else (editor prefs, window size, background images,
// skipped update version) is preserved to avoid wiping unrelated settings.
func (a *App) SaveConfig(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}

	// Update startup setting if changed
	if cfg.Startup.LaunchOnStartup != a.config.Startup.LaunchOnStartup {
		if err := config.SetStartupEnabled(cfg.Startup.LaunchOnStartup); err != nil {
			return err
		}
	}

	// Update hotkeys if changed
	hotkeysChanged := cfg.Hotkeys.Fullscreen != a.config.Hotkeys.Fullscreen ||
		cfg.Hotkeys.Region != a.config.Hotkeys.Region ||
		cfg.Hotkeys.Window != a.config.Hotkeys.Window

	// Merge modal-managed groups into the live config
	a.config.Hotkeys = cfg.Hotkeys
	a.config.Startup = cfg.Startup
	a.config.QuickSave = cfg.QuickSave
	a.config.Export = cfg.Export
	a.config.Update.CheckOnStartup = cfg.Update.CheckOnStartup
	// a.config.Update.SkippedVersion intentionally preserved

	// Save to disk
	if err := a.config.Save(); err != nil {
		return err
	}

	// Re-register hotkeys if they changed
	if hotkeysChanged {
		a.hotkeyManager.UnregisterAll()
		a.registerHotkeysFromConfig()
	}

	return nil
}

// SaveLanguage persists the UI language ("zh" or "en") and syncs the tray menu
func (a *App) SaveLanguage(language string) error {
	if language != "zh" && language != "en" {
		return fmt.Errorf("unsupported language: %s", language)
	}
	a.config.Language = language
	if a.trayIcon != nil {
		a.trayIcon.SetLanguage(language)
	}
	return a.config.Save()
}

// SelectFolder opens a folder selection dialog
func (a *App) SelectFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Save Folder",
	})
}

// registerHotkeysFromConfig registers hotkeys based on current config
func (a *App) registerHotkeysFromConfig() {
	// Parse and register fullscreen hotkey
	if mods, key, ok := hotkeys.ParseHotkeyString(a.config.Hotkeys.Fullscreen); ok {
		a.hotkeyManager.Register(hotkeys.HotkeyFullscreen, mods, key)
	}

	// Parse and register region hotkey
	if mods, key, ok := hotkeys.ParseHotkeyString(a.config.Hotkeys.Region); ok {
		a.hotkeyManager.Register(hotkeys.HotkeyRegion, mods, key)
	}

	// Parse and register window hotkey
	if mods, key, ok := hotkeys.ParseHotkeyString(a.config.Hotkeys.Window); ok {
		a.hotkeyManager.Register(hotkeys.HotkeyWindow, mods, key)
	}
}

// GetBackgroundImages returns the list of saved background images (base64 data URLs)
func (a *App) GetBackgroundImages() []string {
	if a.config == nil || a.config.BackgroundImages == nil {
		return []string{}
	}
	return a.config.BackgroundImages
}

// SaveBackgroundImages saves the list of background images to persistent config
func (a *App) SaveBackgroundImages(images []string) error {
	if a.config == nil {
		return nil
	}
	a.config.BackgroundImages = images
	return a.config.Save()
}

// OpenImage opens a file dialog to select an image and returns it as a CaptureResult
func (a *App) OpenImage() (*screenshot.CaptureResult, error) {
	// Show open file dialog
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Image",
		Filters: []runtime.FileFilter{
			{DisplayName: "Image Files", Pattern: "*.png;*.jpg;*.jpeg;*.gif;*.bmp;*.webp"},
			{DisplayName: "PNG Images", Pattern: "*.png"},
			{DisplayName: "JPEG Images", Pattern: "*.jpg;*.jpeg"},
			{DisplayName: "GIF Images", Pattern: "*.gif"},
			{DisplayName: "BMP Images", Pattern: "*.bmp"},
			{DisplayName: "WebP Images", Pattern: "*.webp"},
		},
	})

	if err != nil {
		return nil, err
	}

	if filePath == "" {
		return nil, nil // User cancelled
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Decode image to get dimensions
	var img image.Image
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".png":
		img, err = png.Decode(bytes.NewReader(data))
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(bytes.NewReader(data))
	case ".gif":
		img, err = gif.Decode(bytes.NewReader(data))
	case ".bmp":
		img, err = bmp.Decode(bytes.NewReader(data))
	case ".webp":
		img, err = webp.Decode(bytes.NewReader(data))
	default:
		// Try to decode as any supported format
		img, _, err = image.Decode(bytes.NewReader(data))
	}

	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()

	// Re-encode as PNG for consistent handling in frontend
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	// Return as base64 encoded PNG
	return &screenshot.CaptureResult{
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Data:   base64.StdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}

// GetClipboardImage reads an image from the Windows clipboard
func (a *App) GetClipboardImage() (*screenshot.CaptureResult, error) {
	return screenshot.GetClipboardImage()
}

// CheckForUpdate checks GitHub for a newer version
func (a *App) CheckForUpdate(currentVersion string) (*updater.UpdateInfo, error) {
	return updater.CheckForUpdate(currentVersion)
}

// OpenURL opens a URL in the default browser
func (a *App) OpenURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

// SetSkippedVersion sets a version to skip for update notifications
func (a *App) SetSkippedVersion(version string) error {
	a.config.Update.SkippedVersion = version
	return a.config.Save()
}

// GetSkippedVersion returns the version that user chose to skip
func (a *App) GetSkippedVersion() string {
	return a.config.Update.SkippedVersion
}

// GetEditorConfig returns the current editor panel settings
func (a *App) GetEditorConfig() *config.EditorConfig {
	return &a.config.Editor
}

// SaveEditorConfig saves editor panel settings to persistent config
func (a *App) SaveEditorConfig(editor *config.EditorConfig) error {
	if editor == nil {
		return nil
	}
	a.config.Editor = *editor
	return a.config.Save()
}

// ==================== Screenshot Library ====================

// GetLibraryImages returns all screenshots from QuickSave folder
func (a *App) GetLibraryImages() ([]library.LibraryImage, error) {
	folder := a.config.QuickSave.Folder
	if folder == "" {
		// Fallback to default location
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		folder = filepath.Join(homeDir, "Pictures", "OneSnap")
	}

	opts := library.DefaultScanOptions()
	return library.ScanFolder(folder, opts)
}

// OpenInEditor loads an image file into the editor
// Security: validates path is within QuickSave folder
func (a *App) OpenInEditor(imagePath string) (*screenshot.CaptureResult, error) {
	// Validate path is within QuickSave folder (prevent directory traversal)
	folder := a.config.QuickSave.Folder
	if folder == "" {
		homeDir, _ := os.UserHomeDir()
		folder = filepath.Join(homeDir, "Pictures", "OneSnap")
	}

	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	absFolder, err := filepath.Abs(folder)
	if err != nil {
		return nil, fmt.Errorf("invalid folder path: %w", err)
	}

	// Security check: ensure file is within QuickSave folder
	if !strings.HasPrefix(absPath, absFolder+string(filepath.Separator)) {
		return nil, fmt.Errorf("access denied: file outside QuickSave folder")
	}

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Decode image
	var img image.Image
	ext := strings.ToLower(filepath.Ext(absPath))

	switch ext {
	case ".png":
		img, err = png.Decode(bytes.NewReader(data))
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(bytes.NewReader(data))
	default:
		img, _, err = image.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := img.Bounds()

	// Re-encode as PNG for consistent handling in frontend
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	return &screenshot.CaptureResult{
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Data:   base64.StdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}

// DeleteScreenshot removes a screenshot file from disk
// Security: validates path is within QuickSave folder
func (a *App) DeleteScreenshot(imagePath string) error {
	// Validate path is within QuickSave folder (prevent directory traversal)
	folder := a.config.QuickSave.Folder
	if folder == "" {
		homeDir, _ := os.UserHomeDir()
		folder = filepath.Join(homeDir, "Pictures", "OneSnap")
	}

	absPath, err := filepath.Abs(imagePath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	absFolder, err := filepath.Abs(folder)
	if err != nil {
		return fmt.Errorf("invalid folder path: %w", err)
	}

	// Security check: ensure file is within QuickSave folder
	if !strings.HasPrefix(absPath, absFolder+string(filepath.Separator)) {
		return fmt.Errorf("access denied: file outside QuickSave folder")
	}

	return os.Remove(absPath)
}

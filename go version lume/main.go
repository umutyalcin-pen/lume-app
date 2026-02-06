package main

import (
	"context"
	"fmt"
	"lume-go/internal/config"
	"lume-go/internal/logger"
	"lume-go/internal/metadata"
	"lume-go/internal/organizer"
	"lume-go/internal/validator"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

// Elite Constants (Audit Point 1)
const (
	AppVersion       = "2.1"
	MaxFilesLimit    = 10000
	MaxErrorsDisplay = 10
)

type OrganizeResult struct {
	Success bool
	File    string
	Size    int64 // Elite v2.1: Efficiency Fix
	Error   error
}

type LumeUI struct {
	MainWindow     *walk.MainWindow
	TargetLabel    *walk.Label
	StartBtn       *walk.PushButton
	StatusLabel    *walk.Label
	ThemeBtn       *walk.PushButton
	LangBtn        *walk.PushButton
	ArchiveHeader  *walk.Label
	TargetHeader   *walk.Label
	SelectionLabel *walk.Label
	
	TargetFolder string
	FileCount    int
	FilesToMove  []metadata.FileInfo
	Config       config.Config

	GroupBox       *walk.GroupBox
	SelectBtn      *walk.PushButton
	ProgressBar    *walk.ProgressBar
	CancelBtn      *walk.PushButton
	
	cancelFunc     context.CancelFunc
	mutex          sync.Mutex
	isProcessing   bool
}

var i18n = map[string]map[string]string{
	"tr": {
		"title":          "Lume v2.1 (Precision)",
		"theme_light":    "Aydınlık Mod", "theme_dark": "Karanlık Mod",
		"lang_switch":    "EN", "archive_ops": "Arşiv İşlemleri",
		"target_folder":  "Hedef Klasör:", "not_selected": "Seçilmedi",
		"select_btn":     "Seç...", "drag_drop": "Dosyaları Pencereye Sürükle & Bırak",
		"files_ready":    "%d dosya hazır", "start_btn": "Düzenlemeyi Başlat",
		"warn_title":     "Uyarı", "warn_select": "Lütfen önce bir hedef klasör seçin.",
		"warn_max":       "Maksimum %d dosya eklenebilir.", "success_title": "İşlem Tamamlandı",
		"success_msg":    "%d dosya arşivlendi. %d hata oluştu.", "organizing": "Düzenleniyor...",
		"complete":       "Arşivleme tamamlandı!", "cancel_btn": "İptal",
		"err_val":        "Kontrol hatası: %v", "err_disk": "Yetersiz disk alanı.",
		"proc_count":     "%d / %d dosya işlendi", "cancelled": "İşlem iptal edildi.",
		"err_report":     "Hata Detayları:\n\n%s", "err_same_path": "Kaynak ve hedef aynı olamaz.",
		"checking_space": "Disk alanı kontrol ediliyor...",
		"stats_info":     "Ömür Boyu: %d dosya | %d MB | %d işlem",
	},
	"en": {
		"title":          "Lume v2.1 (Precision)",
		"theme_light":    "Light Mode", "theme_dark": "Dark Mode",
		"lang_switch":    "TR", "archive_ops": "Archive Operations",
		"target_folder":  "Target Folder:", "not_selected": "Not Selected",
		"select_btn":     "Select...", "drag_drop": "Drag & Drop Files Anywhere in Window",
		"files_ready":    "%d files ready", "start_btn": "Start Organizing",
		"warn_title":     "Warning", "warn_select": "Please select a target folder first.",
		"warn_max":       "Maximum %d files allowed.", "success_title": "Processing Complete",
		"success_msg":    "%d files archived. %d errors occurred.", "organizing": "Organizing...",
		"complete":       "Archiving complete!", "cancel_btn": "Cancel",
		"err_val":        "Validation error: %v", "err_disk": "Insufficient disk space.",
		"proc_count":     "%d / %d files processed", "cancelled": "Operation cancelled.",
		"err_report":     "Error Details:\n\n%s", "err_same_path": "Source and target folder are identical.",
		"checking_space": "Checking disk space...",
		"stats_info":     "Lifetime: %d files | %d MB | %d ops",
	},
}

func (ui *LumeUI) T(k string) string { return i18n[ui.Config.Language][k] }

func main() {
	if err := logger.Init(); err != nil { fmt.Printf("Fatal: %v\n", err) }
	
	defer func() {
		if r := recover(); r != nil { logger.Error("Elite Recovery: %v", r) }
		logger.Close()
	}()

	ui := &LumeUI{Config: config.LoadConfig()}

	// Elite Signal Handler Fixed (Audit 2.1 Point 3)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sc
		logger.Info("Shutdown signal received. Shutting down gracefully...")
		ui.mutex.Lock()
		if ui.cancelFunc != nil { ui.cancelFunc() }
		ui.mutex.Unlock()
		logger.Close() // Ensure log is closed
		os.Exit(0)
	}()

	if err := (MainWindow{
		AssignTo: &ui.MainWindow, Title: ui.T("title"), MinSize: Size{420, 450}, Layout: VBox{}, OnDropFiles: ui.HandleDrop,
		Children: []Widget{
			Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{HSpacer{}, PushButton{AssignTo: &ui.LangBtn, Text: ui.T("lang_switch"), OnClicked: ui.ToggleLanguage}, PushButton{AssignTo: &ui.ThemeBtn, Text: ui.GetThemeBtnText(), OnClicked: ui.ToggleTheme}}},
			Label{AssignTo: &ui.ArchiveHeader, Text: ui.T("archive_ops"), Font: Font{PointSize: 10, Bold: true}},
			GroupBox{AssignTo: &ui.GroupBox, Layout: VBox{}, Children: []Widget{
				Composite{Layout: HBox{}, Children: []Widget{Label{AssignTo: &ui.TargetHeader, Text: ui.T("target_folder")}, Label{AssignTo: &ui.TargetLabel, Text: ui.T("not_selected"), TextAlignment: AlignFar}, PushButton{AssignTo: &ui.SelectBtn, Text: ui.T("select_btn"), OnClicked: ui.SelectFolder}}},
				Label{AssignTo: &ui.SelectionLabel, Text: ui.T("drag_drop"), Font: Font{PointSize: 12, Bold: true}},
				Label{AssignTo: &ui.StatusLabel, Text: ui.GetStatusText()},
				ProgressBar{AssignTo: &ui.ProgressBar, MinValue: 0, MaxValue: 100, Visible: false},
			}},
			Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{PushButton{AssignTo: &ui.StartBtn, Text: ui.T("start_btn"), OnClicked: ui.StartOrganizing}, PushButton{AssignTo: &ui.CancelBtn, Text: ui.T("cancel_btn"), Visible: false, OnClicked: ui.CancelOrganizing}}},
		},
	}.Create()); err != nil { panic(err) }
	
	if ui.Config.TargetFolder != "" { ui.TargetFolder = ui.Config.TargetFolder; ui.TargetLabel.SetText(filepath.Base(ui.TargetFolder)) }
	if icon, err := walk.NewIconFromFile("lume.ico"); err == nil { ui.MainWindow.SetIcon(icon) }
	ui.ApplyTheme(); ui.MainWindow.Run()
}

func (ui *LumeUI) GetStatusText() string {
	if ui.FileCount > 0 {
		return fmt.Sprintf(ui.T("files_ready"), ui.FileCount)
	}
	// Display Stats when idle (Audit 2.1 Point 5)
	if ui.Config.Stats.TotalFiles > 0 {
		mb := ui.Config.Stats.TotalSize / (1024 * 1024)
		return fmt.Sprintf(ui.T("stats_info"), ui.Config.Stats.TotalFiles, mb, ui.Config.Stats.TotalOrganized)
	}
	return fmt.Sprintf(ui.T("files_ready"), 0)
}

func (ui *LumeUI) ToggleTheme() { ui.Config.DarkMode = !ui.Config.DarkMode; config.SaveConfig(ui.Config); ui.ThemeBtn.SetText(ui.GetThemeBtnText()); ui.ApplyTheme() }
func (ui *LumeUI) GetThemeBtnText() string { if ui.Config.DarkMode { return ui.T("theme_light") }; return ui.T("theme_dark") }
func (ui *LumeUI) ToggleLanguage() { if ui.Config.Language == "tr" { ui.Config.Language = "en" } else { ui.Config.Language = "tr" }; config.SaveConfig(ui.Config); ui.RefreshLocalization() }
func (ui *LumeUI) RefreshLocalization() { ui.MainWindow.SetTitle(ui.T("title")); ui.LangBtn.SetText(ui.T("lang_switch")); ui.ThemeBtn.SetText(ui.GetThemeBtnText()); ui.ArchiveHeader.SetText(ui.T("archive_ops")); ui.TargetHeader.SetText(ui.T("target_folder")); if ui.TargetFolder == "" { ui.TargetLabel.SetText(ui.T("not_selected")) }; ui.SelectBtn.SetText(ui.T("select_btn")); ui.SelectionLabel.SetText(ui.T("drag_drop")); ui.StatusLabel.SetText(ui.GetStatusText()); ui.StartBtn.SetText(ui.T("start_btn")); ui.CancelBtn.SetText(ui.T("cancel_btn")) }
func (ui *LumeUI) ApplyTheme() { bg, tx := walk.Color(walk.RGB(240, 240, 240)), walk.Color(walk.RGB(0, 0, 0)); if ui.Config.DarkMode { bg, tx = walk.Color(walk.RGB(35, 35, 35)), walk.Color(walk.RGB(255, 255, 255)) }; br, _ := walk.NewSolidColorBrush(bg); ui.MainWindow.SetBackground(br); for i := 0; i < ui.MainWindow.Children().Len(); i++ { ui.recursiveStyle(ui.MainWindow.Children().At(i), br, tx) }; ui.MainWindow.Invalidate() }
func (ui *LumeUI) recursiveStyle(w walk.Widget, b walk.Brush, t walk.Color) { w.SetBackground(b); if l, ok := w.(*walk.Label); ok { l.SetTextColor(t) }; if c, ok := w.(walk.Container); ok { for i := 0; i < c.Children().Len(); i++ { ui.recursiveStyle(c.Children().At(i), b, t) } } }
func (ui *LumeUI) SelectFolder() { ui.mutex.Lock(); if ui.isProcessing { ui.mutex.Unlock(); return }; ui.mutex.Unlock(); dlg := new(walk.FileDialog); if ok, _ := dlg.ShowBrowseFolder(ui.MainWindow); ok { if err := validator.CheckWritability(dlg.FilePath); err != nil { walk.MsgBox(ui.MainWindow, ui.T("warn_title"), fmt.Sprintf(ui.T("err_val"), err), walk.MsgBoxIconError); return }; ui.TargetFolder = dlg.FilePath; ui.TargetLabel.SetText(filepath.Base(ui.TargetFolder)); ui.Config.TargetFolder = ui.TargetFolder; config.SaveConfig(ui.Config) } }
func (ui *LumeUI) HandleDrop(ps []string) { ui.mutex.Lock(); defer ui.mutex.Unlock(); if ui.isProcessing { return }; for _, p := range ps { if ui.FileCount >= MaxFilesLimit { walk.MsgBox(ui.MainWindow, ui.T("warn_title"), fmt.Sprintf(ui.T("warn_max"), MaxFilesLimit), walk.MsgBoxIconWarning); break }; if !validator.IsPathSafe(p) { continue }; info, err := metadata.GetFileInfo(p); if err != nil { logger.Error("Drop check err: %v", err); continue }; if filepath.Dir(info.Path) == ui.TargetFolder { continue }; ui.FilesToMove = append(ui.FilesToMove, info); ui.FileCount++ }; ui.StatusLabel.SetText(ui.GetStatusText()) }
func (ui *LumeUI) StartOrganizing() { ui.mutex.Lock(); if ui.TargetFolder == "" { ui.mutex.Unlock(); walk.MsgBox(ui.MainWindow, ui.T("warn_title"), ui.T("warn_select"), walk.MsgBoxIconWarning); return }; if len(ui.FilesToMove) == 0 || ui.isProcessing { ui.mutex.Unlock(); return }; ui.mutex.Unlock(); ui.StatusLabel.SetText(ui.T("checking_space")); var ts int64; for _, f := range ui.FilesToMove { ts += f.Size }; if err := validator.CheckDiskSpace(ui.TargetFolder, ts); err != nil { walk.MsgBox(ui.MainWindow, ui.T("warn_title"), fmt.Sprintf("%s (%v)", ui.T("err_disk"), err), walk.MsgBoxIconError); return }; ui.mutex.Lock(); ui.isProcessing = true; ui.mutex.Unlock(); ui.StartBtn.SetEnabled(false); ui.CancelBtn.SetVisible(true); ui.ProgressBar.SetVisible(true); ui.ProgressBar.SetValue(0); ctx, cancel := context.WithCancel(context.Background()); ui.cancelFunc = cancel; go func() { defer cancel(); ui.mutex.Lock(); wl, target := ui.FilesToMove, ui.TargetFolder; ui.mutex.Unlock(); total, res, successCount := len(wl), make([]OrganizeResult, 0), 0; for i, info := range wl { select { case <-ctx.Done(): ui.MainWindow.Synchronize(func() { ui.StatusLabel.SetText(ui.T("cancelled")) }); goto finish; default: err := organizer.MoveFile(info, target)
				if err == nil { successCount++; res = append(res, OrganizeResult{Success: true, File: info.Filename, Size: info.Size}) } else { res = append(res, OrganizeResult{Success: false, File: info.Filename, Size: info.Size, Error: err}) }
				pr := (i + 1) * 100 / total
				ui.MainWindow.Synchronize(func() { ui.ProgressBar.SetValue(pr); ui.StatusLabel.SetText(fmt.Sprintf(ui.T("proc_count"), i+1, total)) })
			}
		}
	finish:
		// Enhanced Stats Logic (Audit 2.1 Points 1 & 2)
		ui.mutex.Lock()
		if successCount > 0 {
			ui.Config.Stats.TotalFiles += successCount
			ui.Config.Stats.TotalOrganized++
			for _, r := range res { if r.Success { ui.Config.Stats.TotalSize += r.Size } }
			config.SaveConfig(ui.Config)
		}
		ui.mutex.Unlock()

		ui.MainWindow.Synchronize(func() {
			ec := total - successCount; if ec < 0 { ec = 0 }
			sm := fmt.Sprintf(ui.T("success_msg"), successCount, ec)
			if ec > 0 {
				var report string; lim := 0; for _, r := range res { if !r.Success { report += fmt.Sprintf("- %s: %v\n", r.File, r.Error); lim++; if lim > MaxErrorsDisplay { report += "...see log"; break } } }; walk.MsgBox(ui.MainWindow, ui.T("success_title"), sm+"\n\n"+fmt.Sprintf(ui.T("err_report"), report), walk.MsgBoxIconWarning)
			} else if successCount > 0 { walk.MsgBox(ui.MainWindow, ui.T("success_title"), sm, walk.MsgBoxIconInformation) }
			ui.mutex.Lock(); ui.FilesToMove, ui.FileCount, ui.isProcessing = nil, 0, false; ui.mutex.Unlock(); ui.StartBtn.SetEnabled(true); ui.CancelBtn.SetVisible(false); ui.ProgressBar.SetVisible(false); ui.StatusLabel.SetText(ui.GetStatusText())
		})
	}()
}

func (ui *LumeUI) CancelOrganizing() { ui.mutex.Lock(); defer ui.mutex.Unlock(); if ui.cancelFunc != nil { ui.cancelFunc() } }

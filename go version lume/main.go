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
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

const (
	AppVersion       = "2.1"
	MaxFilesLimit    = 10000
	MaxErrorsDisplay = 10
)

type OrganizeResult struct {
	Success bool
	File    string
	Size    int64
	Error   error
}

type FileItem struct {
	Index  int
	Name   string
	Size   string
	Source string
	Target string
	Status string
}

type FileModel struct {
	walk.TableModelBase
	items []*FileItem
	ui    *LumeUI
}

func (m *FileModel) StyleCell(style *walk.CellStyle) {
	if m.ui != nil && m.ui.Config.DarkMode {
		style.BackgroundColor = walk.Color(walk.RGB(35, 35, 35))
		style.TextColor = walk.Color(walk.RGB(255, 255, 255))
	} else {
		if style.Row()%2 == 1 {
			style.BackgroundColor = walk.Color(walk.RGB(240, 240, 240))
		} else {
			style.BackgroundColor = walk.Color(walk.RGB(255, 255, 255))
		}
		style.TextColor = walk.Color(walk.RGB(0, 0, 0))
	}
}

func (m *FileModel) RowCount() int {
	return len(m.items)
}

func (m *FileModel) Value(row, col int) interface{} {
	item := m.items[row]
	switch col {
	case 0:
		return item.Name
	case 1:
		return item.Size
	case 2:
		return item.Source
	case 3:
		return item.Target
	case 4:
		return item.Status
	}
	return nil
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

	GroupBox    *walk.GroupBox
	SelectBtn   *walk.PushButton
	ProgressBar *walk.ProgressBar
	CancelBtn   *walk.PushButton
	DryRunCheck *walk.CheckBox
	RenameCheck *walk.CheckBox

	FilesTable *walk.TableView
	FilesModel *FileModel

	cancelFunc   context.CancelFunc
	mutex        sync.Mutex
	isProcessing bool
}

var i18n = map[string]map[string]string{
	"tr": {
		"title":       "Lume v2.1",
		"theme_light": "Aydınlık Mod", "theme_dark": "Karanlık Mod",
		"lang_switch": "EN", "archive_ops": "Arşiv İşlemleri",
		"target_folder": "Hedef Klasör:", "not_selected": "Seçilmedi",
		"select_btn": "Seç...", "drag_drop": "Dosyaları Pencereye Sürükle ve Bırak", "files_ready": "%d dosya hazır", "start_btn": "Düzenlemeyi Başlat",
		"warn_title": "Uyarı", "warn_select": "Lütfen önce bir hedef klasör seçin.",
		"warn_max": "Maksimum %d dosya eklenebilir.", "success_title": "İşlem Tamamlandı",
		"success_msg": "%d dosya arşivlendi. %d hata oluştu.", "organizing": "Düzenleniyor...",
		"complete": "Arşivleme tamamlandı!", "cancel_btn": "İptal",
		"err_val": "Kontrol hatası: %v", "err_disk": "Yetersiz disk alanı.",
		"proc_count": "%d / %d dosya işlendi", "cancelled": "İşlem iptal edildi.",
		"err_report": "Hata Detayları:\n\n%s", "err_same_path": "Kaynak ve hedef aynı olamaz.",
		"checking_space": "Disk alanı kontrol ediliyor...",
		"stats_info":     "Şimdiye Kadar: %d dosya | %d MB | %d işlem",
		"col_name":       "Dosya Adı", "col_size": "Boyut",
		"col_source": "Kaynak", "col_target": "Hedef Klasör",
		"col_status":         "Durum",
		"dry_run_label":      "Sadece Simüle Et (Kopyalama Yapma)",
		"rename_label":       "Çekim Tarihine Göre Yeniden Adlandır",
		"sim_complete_title": "Simülasyon Tamamlandı",
		"sim_complete_msg":   "Simülasyon başarıyla tamamlandı!\nKopyalanacak: %d dosya\nAtlanacak (Kopya): %d dosya\nDiske hiçbir veri yazılmadı.",
		"drag_target_ask":    "Sürüklediğiniz klasörü HEDEF klasör olarak ayarlamak ister misiniz?\n\nEvet: Hedef Klasör Yap\nHayır: Kaynak Klasör Olarak Tara",
		"warn_system_dir":    "Korumalı sistem dizini hedef olarak seçilemez.",
		"warn_write":         "Seçilen klasöre yazma izniniz yok: %v",
		"status_ready":       "Hazır",
		"status_would_ok":    "Kopyalanacak (Test)",
		"status_would_skip":  "Atlanacak (Kopya)",
		"status_ok":          "Tamamlandı",
		"status_skipped":     "Atlandı (Kopya)",
		"status_err":         "Hata",
	},
	"en": {
		"title":       "Lume v2.1",
		"theme_light": "Light Mode", "theme_dark": "Dark Mode",
		"lang_switch": "TR", "archive_ops": "Archive Operations",
		"target_folder": "Target Folder:", "not_selected": "Not Selected",
		"select_btn": "Select...", "drag_drop": "Drag and Drop Files Anywhere in the Window",
		"files_ready": "%d files ready", "start_btn": "Start Organizing",
		"warn_title": "Warning", "warn_select": "Please select a target folder first.",
		"warn_max": "Maximum %d files allowed.", "success_title": "Processing Complete",
		"success_msg": "%d files archived. %d errors occurred.", "organizing": "Organizing...",
		"complete": "Archiving complete!", "cancel_btn": "Cancel",
		"err_val": "Validation error: %v", "err_disk": "Insufficient disk space.",
		"proc_count": "%d / %d files processed", "cancelled": "Operation cancelled.",
		"err_report": "Error Details:\n\n%s", "err_same_path": "Source and target folders are identical.",
		"checking_space": "Checking disk space...",
		"stats_info":     "So Far: %d files | %d MB | %d ops",
		"col_name":       "File Name", "col_size": "Size",
		"col_source": "Source", "col_target": "Target Folder",
		"col_status":         "Status",
		"dry_run_label":      "Simulate Only (Do Not Copy)",
		"rename_label":       "Rename to Shooting Date (YYYYMMDD_HHMMSS)",
		"sim_complete_title": "Simulation Complete",
		"sim_complete_msg":   "Simulation completed successfully!\nTo copy: %d files\nTo skip (Dup): %d files\nNo files were written to disk.",
		"drag_target_ask":    "Do you want to set the dragged folder as the TARGET folder?\n\nYes: Set as Target Folder\nNo: Scan as Source Folder",
		"warn_system_dir":    "Protected system directory cannot be set as target.",
		"warn_write":         "Cannot write to selected folder: %v",
		"status_ready":       "Ready",
		"status_would_ok":    "Would Copy (Test)",
		"status_would_skip":  "Would Skip (Dup)",
		"status_ok":          "Completed",
		"status_skipped":     "Skipped (Dup)",
		"status_err":         "Error",
	},
}

func getProcByOrdinal(dllName string, ordinal uintptr) uintptr {
	dll, err := syscall.LoadLibrary(dllName)
	if err != nil {
		return 0
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getProcAddress := kernel32.NewProc("GetProcAddress")
	addr, _, _ := getProcAddress.Call(uintptr(dll), ordinal)
	return addr
}

var (
	fnSetPreferredAppMode    = getProcByOrdinal("uxtheme.dll", 135)
	fnAllowDarkModeForWindow = getProcByOrdinal("uxtheme.dll", 133)
	fnFlushMenuThemes        = getProcByOrdinal("uxtheme.dll", 136)

	dwmapi                    = syscall.NewLazyDLL("dwmapi.dll")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

func initWinThemes() {
	if fnSetPreferredAppMode != 0 {

		syscall.Syscall(fnSetPreferredAppMode, 1, uintptr(1), 0, 0)
	}
}

func enableDarkModeForHWnd(hwnd win.HWND, enabled bool) {
	var val uintptr = 0
	if enabled {
		val = 1
	}
	if fnAllowDarkModeForWindow != 0 {
		syscall.Syscall(fnAllowDarkModeForWindow, 2, uintptr(hwnd), val, 0)
	}

	if procDwmSetWindowAttribute.Find() == nil {
		var value int32 = 0
		if enabled {
			value = 1
		}

		procDwmSetWindowAttribute.Call(
			uintptr(hwnd),
			uintptr(20),
			uintptr(unsafe.Pointer(&value)),
			unsafe.Sizeof(value),
		)
	}
}

func (ui *LumeUI) T(k string) string { return i18n[ui.Config.Language][k] }

func main() {
	initWinThemes()
	if err := logger.Init(); err != nil {
		fmt.Printf("Fatal: %v\n", err)
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Error("Elite Recovery: %v", r)
		}
		logger.Close()
	}()

	ui := &LumeUI{
		Config:     config.LoadConfig(),
		FilesModel: new(FileModel),
	}
	ui.FilesModel.ui = ui

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sc
		logger.Info("Shutdown signal received. Cancelling active operations...")
		ui.mutex.Lock()
		if ui.cancelFunc != nil {
			ui.cancelFunc()
		}
		ui.mutex.Unlock()

	}()

	if err := (MainWindow{
		AssignTo: &ui.MainWindow, Title: ui.T("title"), MinSize: Size{Width: 520, Height: 500}, Layout: VBox{Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10}}, OnDropFiles: ui.HandleDrop,
		Children: []Widget{
			Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{HSpacer{}, PushButton{AssignTo: &ui.LangBtn, Text: ui.T("lang_switch"), OnClicked: ui.ToggleLanguage}, PushButton{AssignTo: &ui.ThemeBtn, Text: ui.GetThemeBtnText(), OnClicked: ui.ToggleTheme}}},
			Label{AssignTo: &ui.ArchiveHeader, Text: ui.T("archive_ops"), Font: Font{PointSize: 10, Bold: true}},
			GroupBox{AssignTo: &ui.GroupBox, Layout: VBox{}, Children: []Widget{
				Composite{Layout: HBox{}, Children: []Widget{Label{AssignTo: &ui.TargetHeader, Text: ui.T("target_folder")}, Label{AssignTo: &ui.TargetLabel, Text: ui.T("not_selected"), TextAlignment: AlignFar}, PushButton{AssignTo: &ui.SelectBtn, Text: ui.T("select_btn"), OnClicked: ui.SelectFolder}}},
				Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{
					CheckBox{AssignTo: &ui.DryRunCheck, Text: ui.T("dry_run_label"), Checked: ui.Config.DryRun, OnCheckedChanged: ui.SaveConfigState},
					HSpacer{},
					CheckBox{AssignTo: &ui.RenameCheck, Text: ui.T("rename_label"), Checked: ui.Config.Rename, OnCheckedChanged: ui.SaveConfigState},
				}},
				Label{AssignTo: &ui.SelectionLabel, Text: ui.T("drag_drop"), Font: Font{PointSize: 11, Bold: true}},
				Label{AssignTo: &ui.StatusLabel, Text: ui.GetStatusText()},
				ProgressBar{AssignTo: &ui.ProgressBar, MinValue: 0, MaxValue: 100, Visible: false},
			}},
			TableView{
				AssignTo:   &ui.FilesTable,
				Model:      ui.FilesModel,
				MinSize:    Size{Width: 250, Height: 180},
				CellStyler: ui.FilesModel,
				Columns: []TableViewColumn{
					{Title: ui.T("col_name"), Width: 150},
					{Title: ui.T("col_size"), Width: 65},
					{Title: ui.T("col_source"), Width: 75},
					{Title: ui.T("col_target"), Width: 110},
					{Title: ui.T("col_status"), Width: 100},
				},
			},
			Composite{Layout: HBox{MarginsZero: true}, Children: []Widget{PushButton{AssignTo: &ui.StartBtn, Text: ui.T("start_btn"), OnClicked: ui.StartOrganizing}, PushButton{AssignTo: &ui.CancelBtn, Text: ui.T("cancel_btn"), Visible: false, OnClicked: ui.CancelOrganizing}}},
		},
	}.Create()); err != nil {
		panic(err)
	}

	if ui.Config.TargetFolder != "" {
		ui.TargetFolder = ui.Config.TargetFolder
		ui.updateTargetLabel()
	}
	var icon *walk.Icon
	var err error
	for _, id := range []int{3, 7, 11, 2, 1} {
		if icon, err = walk.NewIconFromResourceId(id); err == nil {
			break
		}
	}
	if err != nil {
		icon, err = walk.NewIconFromFile("lume.ico")
	}
	if err == nil {
		ui.MainWindow.SetIcon(icon)
	}
	ui.MainWindow.Starting().Attach(func() {
		ui.ApplyTheme()
	})
	ui.MainWindow.Run()
}

func (ui *LumeUI) GetStatusText() string {
	if ui.FileCount > 0 {
		return fmt.Sprintf(ui.T("files_ready"), ui.FileCount)
	}

	if ui.Config.Stats.TotalFiles > 0 {
		mb := ui.Config.Stats.TotalSize / (1024 * 1024)
		return fmt.Sprintf(ui.T("stats_info"), ui.Config.Stats.TotalFiles, mb, ui.Config.Stats.TotalOrganized)
	}
	return fmt.Sprintf(ui.T("files_ready"), 0)
}

func (ui *LumeUI) ToggleTheme() {
	ui.Config.DarkMode = !ui.Config.DarkMode
	config.SaveConfig(ui.Config)
	ui.ThemeBtn.SetText(ui.GetThemeBtnText())
	ui.ApplyTheme()
}
func (ui *LumeUI) GetThemeBtnText() string {
	if ui.Config.DarkMode {
		return ui.T("theme_light")
	}
	return ui.T("theme_dark")
}
func (ui *LumeUI) ToggleLanguage() {
	if ui.Config.Language == "tr" {
		ui.Config.Language = "en"
	} else {
		ui.Config.Language = "tr"
	}
	config.SaveConfig(ui.Config)
	ui.RefreshLocalization()
}
func (ui *LumeUI) RefreshLocalization() {
	ui.MainWindow.SetTitle(ui.T("title"))
	ui.LangBtn.SetText(ui.T("lang_switch"))
	ui.ThemeBtn.SetText(ui.GetThemeBtnText())
	ui.ArchiveHeader.SetText(ui.T("archive_ops"))
	ui.TargetHeader.SetText(ui.T("target_folder"))
	ui.updateTargetLabel()
	ui.SelectBtn.SetText(ui.T("select_btn"))
	ui.SelectionLabel.SetText(ui.T("drag_drop"))
	ui.StatusLabel.SetText(ui.GetStatusText())
	ui.StartBtn.SetText(ui.T("start_btn"))
	ui.CancelBtn.SetText(ui.T("cancel_btn"))
	if ui.DryRunCheck != nil {
		ui.DryRunCheck.SetText(ui.T("dry_run_label"))
	}
	if ui.RenameCheck != nil {
		ui.RenameCheck.SetText(ui.T("rename_label"))
	}
	if ui.FilesTable != nil && ui.FilesTable.Columns().Len() > 4 {
		ui.FilesTable.Columns().At(0).SetTitle(ui.T("col_name"))
		ui.FilesTable.Columns().At(1).SetTitle(ui.T("col_size"))
		ui.FilesTable.Columns().At(2).SetTitle(ui.T("col_source"))
		ui.FilesTable.Columns().At(3).SetTitle(ui.T("col_target"))
		ui.FilesTable.Columns().At(4).SetTitle(ui.T("col_status"))
	}
	if ui.FilesModel != nil && len(ui.FilesModel.items) > 0 {
		for i := range ui.FilesModel.items {
			status := ui.FilesModel.items[i].Status
			switch status {
			case "Hazır", "Ready":
				ui.FilesModel.items[i].Status = ui.T("status_ready")
			case "Kopyalanacak (Test)", "Would Copy (Test)":
				ui.FilesModel.items[i].Status = ui.T("status_would_ok")
			case "Atlanacak (Kopya)", "Would Skip (Dup)":
				ui.FilesModel.items[i].Status = ui.T("status_would_skip")
			case "Tamamlandı", "Completed":
				ui.FilesModel.items[i].Status = ui.T("status_ok")
			case "Atlandı (Kopya)", "Skipped (Dup)":
				ui.FilesModel.items[i].Status = ui.T("status_skipped")
			case "Hata", "Error":
				ui.FilesModel.items[i].Status = ui.T("status_err")
			default:
				if strings.HasPrefix(status, "Hata:") || strings.HasPrefix(status, "Error:") {
					idx := strings.Index(status, ":")
					if idx != -1 {
						ui.FilesModel.items[i].Status = ui.T("status_err") + status[idx:]
					}
				}
			}
		}
		ui.FilesModel.PublishRowsReset()
	}
}
func (ui *LumeUI) updateTargetLabel() {
	if ui.TargetFolder == "" {
		ui.TargetLabel.SetText(ui.T("not_selected"))
		return
	}
	base := filepath.Base(ui.TargetFolder)
	if base == "\\" || base == "/" || base == "." || len(ui.TargetFolder) <= 3 {
		ui.TargetLabel.SetText(ui.TargetFolder)
	} else {
		ui.TargetLabel.SetText(base)
	}
}
func (ui *LumeUI) ApplyTheme() {
	if fnSetPreferredAppMode != 0 {
		var mode uintptr = 3
		if ui.Config.DarkMode {
			mode = 2
		}
		syscall.Syscall(fnSetPreferredAppMode, 1, mode, 0, 0)
	}
	if fnFlushMenuThemes != 0 {
		syscall.Syscall(fnFlushMenuThemes, 0, 0, 0, 0)
	}

	bg, tx := walk.Color(walk.RGB(240, 240, 240)), walk.Color(walk.RGB(0, 0, 0))
	var winBg win.COLORREF = win.RGB(240, 240, 240)
	var winTx win.COLORREF = win.RGB(0, 0, 0)
	if ui.Config.DarkMode {
		bg, tx = walk.Color(walk.RGB(35, 35, 35)), walk.Color(walk.RGB(255, 255, 255))
		winBg = win.RGB(35, 35, 35)
		winTx = win.RGB(255, 255, 255)
	}

	if oldBr := ui.MainWindow.Background(); oldBr != nil {
		oldBr.Dispose()
	}
	br, _ := walk.NewSolidColorBrush(bg)
	ui.MainWindow.SetBackground(br)

	enableDarkModeForHWnd(ui.MainWindow.Handle(), ui.Config.DarkMode)

	for i := 0; i < ui.MainWindow.Children().Len(); i++ {
		ui.recursiveStyle(ui.MainWindow.Children().At(i), br, tx)
	}

	if ui.FilesTable != nil {
		enableDarkModeForHWnd(ui.FilesTable.Handle(), ui.Config.DarkMode)

		if ui.Config.DarkMode {
			win.SetWindowTheme(ui.FilesTable.Handle(), syscall.StringToUTF16Ptr("DarkMode_ItemsView"), nil)
		} else {
			win.SetWindowTheme(ui.FilesTable.Handle(), syscall.StringToUTF16Ptr("Explorer"), nil)
		}
		win.SendMessage(ui.FilesTable.Handle(), win.LVM_SETBKCOLOR, 0, uintptr(winBg))
		win.SendMessage(ui.FilesTable.Handle(), win.LVM_SETTEXTCOLOR, 0, uintptr(winTx))
		win.SendMessage(ui.FilesTable.Handle(), win.LVM_SETTEXTBKCOLOR, 0, uintptr(winBg))
		ui.FilesTable.Invalidate()
	}

	ui.MainWindow.Invalidate()
}
func (ui *LumeUI) recursiveStyle(w walk.Widget, b walk.Brush, t walk.Color) {
	w.SetBackground(b)

	enableDarkModeForHWnd(w.Handle(), ui.Config.DarkMode)
	if _, ok := w.(*walk.TableView); ok {
		if ui.Config.DarkMode {
			win.SetWindowTheme(w.Handle(), syscall.StringToUTF16Ptr("DarkMode_ItemsView"), nil)
		} else {
			win.SetWindowTheme(w.Handle(), syscall.StringToUTF16Ptr("Explorer"), nil)
		}
	} else {
		if ui.Config.DarkMode {
			win.SetWindowTheme(w.Handle(), syscall.StringToUTF16Ptr("DarkMode_Explorer"), nil)
		} else {
			win.SetWindowTheme(w.Handle(), syscall.StringToUTF16Ptr("Explorer"), nil)
		}
	}
	if l, ok := w.(*walk.Label); ok {
		l.SetTextColor(t)
	}
	if _, ok := w.(*walk.PushButton); ok {
		w.Invalidate()
	}
	if c, ok := w.(walk.Container); ok {
		for i := 0; i < c.Children().Len(); i++ {
			ui.recursiveStyle(c.Children().At(i), b, t)
		}
	}
	w.Invalidate()
}
func (ui *LumeUI) SelectFolder() {
	ui.mutex.Lock()
	if ui.isProcessing {
		ui.mutex.Unlock()
		return
	}
	ui.mutex.Unlock()
	dlg := new(walk.FileDialog)
	if ok, _ := dlg.ShowBrowseFolder(ui.MainWindow); ok {

		if validator.IsSystemDir(dlg.FilePath) {
			walk.MsgBox(ui.MainWindow, ui.T("warn_title"), "Korumalı sistem dizini hedef olarak seçilemez. / Protected system directory cannot be selected as target.", walk.MsgBoxIconError)
			return
		}
		if err := validator.CheckWritability(dlg.FilePath); err != nil {
			walk.MsgBox(ui.MainWindow, ui.T("warn_title"), fmt.Sprintf(ui.T("err_val"), err), walk.MsgBoxIconError)
			return
		}
		ui.TargetFolder = dlg.FilePath
		ui.updateTargetLabel()
		ui.Config.TargetFolder = ui.TargetFolder
		config.SaveConfig(ui.Config)
	}
}
func (ui *LumeUI) SaveConfigState() {
	ui.Config.DryRun = ui.DryRunCheck.Checked()
	ui.Config.Rename = ui.RenameCheck.Checked()
	config.SaveConfig(ui.Config)
}

func (ui *LumeUI) HandleDrop(ps []string) {
	ui.mutex.Lock()
	if ui.isProcessing {
		ui.mutex.Unlock()
		return
	}
	ui.mutex.Unlock()

	if len(ps) == 1 {
		stat, err := os.Stat(ps[0])
		if err == nil && stat.IsDir() {
			ret := walk.MsgBox(ui.MainWindow, ui.T("title"), ui.T("drag_target_ask"), walk.MsgBoxYesNo|walk.MsgBoxIconQuestion)
			if ret == win.IDYES {
				if validator.IsSystemDir(ps[0]) {
					walk.MsgBox(ui.MainWindow, ui.T("warn_title"), ui.T("warn_system_dir"), walk.MsgBoxIconWarning)
					return
				}
				if err := validator.CheckWritability(ps[0]); err != nil {
					walk.MsgBox(ui.MainWindow, ui.T("warn_title"), fmt.Sprintf(ui.T("warn_write"), err), walk.MsgBoxIconWarning)
					return
				}
				ui.mutex.Lock()
				ui.TargetFolder = ps[0]
				ui.updateTargetLabel()
				ui.Config.TargetFolder = ui.TargetFolder
				config.SaveConfig(ui.Config)
				ui.mutex.Unlock()
				return
			}
		}
	}

	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	var addFile func(string)
	addFile = func(p string) {
		if ui.FileCount >= MaxFilesLimit {
			return
		}
		if !validator.IsPathSafe(p) {
			return
		}

		if validator.IsSystemDir(p) {
			return
		}

		stat, err := os.Lstat(p)
		if err != nil {
			return
		}

		if stat.Mode()&os.ModeSymlink != 0 {
			return
		}

		if stat.IsDir() {
			if ui.TargetFolder != "" && validator.IsNested(p, ui.TargetFolder) {
				return
			}
			_ = filepath.WalkDir(p, func(subPath string, d os.DirEntry, walkErr error) error {
				if ui.FileCount >= MaxFilesLimit {
					return fmt.Errorf("limit reached")
				}
				if walkErr != nil {
					return nil
				}

				if d.Type()&os.ModeSymlink != 0 {
					return nil
				}
				if d.IsDir() {
					return nil
				}
				addFile(subPath)
				return nil
			})
			return
		}

		info, err := metadata.GetFileInfo(p)
		if err != nil {
			logger.Error("Drop check err: %v", err)
			return
		}

		if filepath.Dir(info.Path) == ui.TargetFolder {
			return
		}

		for _, existing := range ui.FilesToMove {
			if existing.Path == info.Path {
				return
			}
		}

		ui.FilesToMove = append(ui.FilesToMove, info)
		ui.FileCount++
	}

	for _, p := range ps {
		if ui.FileCount >= MaxFilesLimit {
			walk.MsgBox(ui.MainWindow, ui.T("warn_title"), fmt.Sprintf(ui.T("warn_max"), MaxFilesLimit), walk.MsgBoxIconWarning)
			break
		}
		addFile(p)
	}

	ui.FilesModel.items = nil
	for i, f := range ui.FilesToMove {
		targetPath := filepath.Join(f.Year, f.Month, f.Device)
		ui.FilesModel.items = append(ui.FilesModel.items, &FileItem{
			Index:  i,
			Name:   f.Filename,
			Size:   formatSize(f.Size),
			Source: f.Source,
			Target: targetPath,
			Status: ui.T("status_ready"),
		})
	}
	ui.FilesModel.PublishRowsReset()

	ui.StatusLabel.SetText(ui.GetStatusText())
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
func (ui *LumeUI) StartOrganizing() {
	ui.mutex.Lock()
	if ui.TargetFolder == "" {
		ui.mutex.Unlock()
		walk.MsgBox(ui.MainWindow, ui.T("warn_title"), ui.T("warn_select"), walk.MsgBoxIconWarning)
		return
	}
	if len(ui.FilesToMove) == 0 || ui.isProcessing {
		ui.mutex.Unlock()
		return
	}

	wl := make([]metadata.FileInfo, len(ui.FilesToMove))
	copy(wl, ui.FilesToMove)
	target := ui.TargetFolder

	ui.FilesToMove = nil
	ui.FileCount = 0
	ui.isProcessing = true
	ui.mutex.Unlock()

	activeFiles := make([]metadata.FileInfo, 0, len(wl))
	for _, f := range wl {
		srcDir := filepath.Dir(f.Path)
		if srcDir == target || validator.IsNested(srcDir, target) {
			continue
		}
		activeFiles = append(activeFiles, f)
	}

	if len(activeFiles) == 0 {
		ui.mutex.Lock()
		ui.isProcessing = false
		ui.mutex.Unlock()
		walk.MsgBox(ui.MainWindow, ui.T("warn_title"), ui.T("err_same_path"), walk.MsgBoxIconWarning)
		return
	}
	wl = activeFiles

	ui.StatusLabel.SetText(ui.T("checking_space"))
	var ts int64
	for _, f := range wl {
		ts += f.Size
	}
	if err := validator.CheckDiskSpace(target, ts); err != nil {
		ui.mutex.Lock()
		ui.isProcessing = false
		ui.mutex.Unlock()
		walk.MsgBox(ui.MainWindow, ui.T("warn_title"), fmt.Sprintf("%s (%v)", ui.T("err_disk"), err), walk.MsgBoxIconError)
		return
	}

	ui.StartBtn.SetEnabled(false)
	ui.CancelBtn.SetVisible(true)
	ui.ProgressBar.SetVisible(true)
	ui.ProgressBar.SetValue(0)

	ctx, cancel := context.WithCancel(context.Background())
	ui.mutex.Lock()
	ui.cancelFunc = cancel
	ui.mutex.Unlock()

	go func() {
		defer cancel()
		total, res, successCount, skipCount := len(wl), make([]OrganizeResult, 0), 0, 0
		dryRun := ui.Config.DryRun
		rename := ui.Config.Rename

		state := organizer.NewState()

		for i, info := range wl {
			select {
			case <-ctx.Done():
				ui.MainWindow.Synchronize(func() {
					ui.StatusLabel.SetText(ui.T("cancelled"))
				})
				goto finish
			default:
				finalPath, skipped, err := organizer.ArchiveFileWithOptions(ctx, info, target, dryRun, rename, state)
				statusStr := ""
				if err == nil {
					if skipped {
						skipCount++
						if dryRun {
							statusStr = ui.T("status_would_skip")
						} else {
							statusStr = ui.T("status_skipped")
						}
					} else {
						successCount++
						if dryRun {
							statusStr = ui.T("status_would_ok")
						} else {
							statusStr = ui.T("status_ok")
						}
					}
					res = append(res, OrganizeResult{Success: true, File: info.Filename, Size: info.Size})
				} else {
					statusStr = fmt.Sprintf("%s: %v", ui.T("status_err"), err)
					res = append(res, OrganizeResult{Success: false, File: info.Filename, Size: info.Size, Error: err})
				}

				ui.MainWindow.Synchronize(func() {
					if i < len(ui.FilesModel.items) {
						ui.FilesModel.items[i].Status = statusStr
						if rename && err == nil {
							ui.FilesModel.items[i].Name = filepath.Base(finalPath)
						}
						ui.FilesModel.PublishRowChanged(i)
					}
					pr := (i + 1) * 100 / total
					ui.ProgressBar.SetValue(pr)
					ui.StatusLabel.SetText(fmt.Sprintf(ui.T("proc_count"), i+1, total))
				})
			}
		}
	finish:

		ui.mutex.Lock()
		ui.cancelFunc = nil
		if !dryRun && successCount > 0 {
			ui.Config.Stats.TotalFiles += successCount
			ui.Config.Stats.TotalOrganized++
			for _, r := range res {
				if r.Success {
					ui.Config.Stats.TotalSize += r.Size
				}
			}
			if err := config.SaveConfig(ui.Config); err != nil {
				logger.Error("Failed to save statistics config: %v", err)
			}
		}
		ui.isProcessing = false
		ui.mutex.Unlock()

		ui.MainWindow.Synchronize(func() {
			if dryRun {
				sm := fmt.Sprintf(ui.T("sim_complete_msg"), successCount, skipCount)
				walk.MsgBox(ui.MainWindow, ui.T("sim_complete_title"), sm, walk.MsgBoxIconInformation)
			} else {
				ec := total - successCount - skipCount
				if ec < 0 {
					ec = 0
				}
				sm := fmt.Sprintf(ui.T("success_msg"), successCount, ec)
				if ec > 0 {
					var report string
					lim := 0
					for _, r := range res {
						if !r.Success {
							report += fmt.Sprintf("- %s: %v\n", r.File, r.Error)
							lim++
							if lim > MaxErrorsDisplay {
								report += "...see log"
								break
							}
						}
					}
					walk.MsgBox(ui.MainWindow, ui.T("success_title"), sm+"\n\n"+fmt.Sprintf(ui.T("err_report"), report), walk.MsgBoxIconWarning)
				} else if successCount > 0 {
					walk.MsgBox(ui.MainWindow, ui.T("success_title"), sm, walk.MsgBoxIconInformation)
				}
			}
			ui.FilesModel.items = nil
			ui.FilesModel.PublishRowsReset()

			ui.StartBtn.SetEnabled(true)
			ui.CancelBtn.SetVisible(false)
			ui.ProgressBar.SetVisible(false)
			ui.StatusLabel.SetText(ui.GetStatusText())
		})
	}()
}

func (ui *LumeUI) CancelOrganizing() {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()
	if ui.cancelFunc != nil {
		ui.cancelFunc()
	}
}

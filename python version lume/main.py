"""
Lume: Smart Photo & Video Archivist
(Super Lightweight Edition)
Security Hardened Version
"""

import os
import time
import threading
import tkinter as tk
from tkinter import ttk, filedialog, messagebox
from pathlib import Path

from tkinterdnd2 import DND_FILES, TkinterDnD

from exif_reader import get_file_info, is_supported_image, get_file_hash
from file_organizer import calculate_new_path, move_file, get_relative_path
from ui_components import DropZone, FileTable, ProgressDialog
import config_manager
from logger_config import logger

from translations import get_text

# Security Constants
MAX_FILES_LIMIT = 10000
MAX_PATH_LENGTH = 260  # Windows MAX_PATH

class LumeApp(TkinterDnD.Tk):
    """Main application class (Super Lightweight - Security Hardened)."""
    
    def __init__(self):
        super().__init__()
        
        # Ayarları yükle
        self.config = config_manager.load_config()
        logger.info("Application started (Lite - Secured)")

        # Pencere ayarları
        self.title("Lume")
        self.lang = self.config.get("language", "en")
        self.is_zen_mode = True
        self.geometry("500x450")
        self.resizable(False, False)
        
        # Tema / Renk Paleti (Zen Light/Dark)
        self.is_dark_mode = self.config.get("appearance_mode") == "dark"
        self._apply_theme_colors()
        
        # Veri depolama
        self.files_data = [] 
        self.added_hashes = set() 
        self.target_folder = self.config.get("target_folder")
        
        # Security: Validate target folder on startup
        if self.target_folder and not self._validate_target_folder(self.target_folder):
            self.target_folder = None
            config_manager.update_setting("target_folder", None)
        
        # UI oluştur
        self._create_ui()
        
        # Eğer kayıtlı klasör varsa etiketi güncelle
        if self.target_folder and os.path.exists(self.target_folder):
            display_path = self._sanitize_path_display(self.target_folder)
            self.folder_path_label.configure(text=display_path)
            
        # Drag-drop bağlantıları
        self._setup_dnd()

    def _validate_target_folder(self, folder_path):
        """Security: Validates target folder for safety."""
        try:
            # Check existence
            if not os.path.exists(folder_path):
                return False
            
            # Check if it's actually a directory
            if not os.path.isdir(folder_path):
                return False
            
            # Resolve real path (detect symlinks/junctions)
            real_path = os.path.realpath(folder_path)
            
            # Check write permissions
            if not os.access(real_path, os.W_OK):
                logger.warning("Target folder not writable")
                return False
            
            # Block system directories using pathlib
            system_paths = [
                Path(os.environ.get('SYSTEMROOT', 'C:\\Windows')),
                Path(os.environ.get('PROGRAMFILES', 'C:\\Program Files')),
                Path(os.environ.get('PROGRAMFILES(X86)', 'C:\\Program Files (x86)')),
            ]
            
            resolved_path = Path(real_path).resolve()
            for sys_path in system_paths:
                try:
                    resolved_path.relative_to(sys_path.resolve())
                    logger.error(f"System directory blocked")
                    return False
                except ValueError:
                    pass  # Not relative to this system path, which is good
            
            return True
            
        except Exception as e:
            logger.error(f"Target folder validation failed: {str(e)}")
            return False

    def _sanitize_path_display(self, path, max_len=40):
        """Security: Sanitizes path for display (privacy)."""
        if len(path) <= max_len:
            return path
        return "..." + path[-(max_len-3):]

    def _apply_theme_colors(self):
        """Renk paletini ayarlar."""
        if self.is_dark_mode:
            self.bg_color = "#121212"
            self.fg_color = "#FFFFFF"
            self.card_color = "#1E1E1E"
            self.accent_color = "#6366f1"
        else:
            self.bg_color = "#F9F9F9"
            self.fg_color = "#333333"
            self.card_color = "#FFFFFF"
            self.accent_color = "#4F46E5"
        
        self.configure(bg=self.bg_color)

    def _create_ui(self):
        """Builds the user interface."""
        
        # Ana container
        self.main_container = tk.Frame(self, bg=self.bg_color)
        self.main_container.pack(fill="both", expand=True, padx=20, pady=20)
        
        # Üst Panel (Başlık + Tema)
        header_frame = tk.Frame(self.main_container, bg=self.bg_color)
        header_frame.pack(fill="x", pady=(0, 20))
        
        # Logo Konteynırı
        logo_container = tk.Frame(header_frame, bg=self.accent_color, padx=5, pady=2)
        logo_container.pack(side="left", padx=(0, 10))

        self.logo_text_label = tk.Label(
            header_frame, 
            text="Lume", 
            font=("Segoe UI", 26, "bold"), 
            bg=self.bg_color, fg=self.accent_color
        )
        self.logo_text_label.pack(side="left")
        
        self.theme_btn = tk.Button(
            header_frame,
            text="☀️" if self.is_dark_mode else "🌙",
            font=("Segoe UI Emoji", 12),
            bg=self.card_color, fg=self.fg_color,
            relief="flat", width=4, height=1,
            command=self._toggle_theme
        )
        self.theme_btn.pack(side="right")

        self.lang_btn = tk.Button(
            header_frame,
            text=self._get_text("lang_name"),
            font=("Segoe UI", 9, "bold"),
            bg=self.card_color, fg=self.fg_color,
            relief="flat", width=4, height=1,
            command=self._toggle_language
        )
        self.lang_btn.pack(side="right", padx=5)
        
        # Sürükle-bırak alanı
        self.drop_zone = DropZone(
            self.main_container, 
            bg_color=self.card_color, 
            active_color="#EEEEEE" if not self.is_dark_mode else "#2A2A2A"
        )
        self.drop_zone.pack(fill="x", pady=(0, 15))
        
        # Zen Mode'da gizli olan kısım
        self.expanded_container = tk.Frame(self.main_container, bg=self.bg_color)
        
        # File table
        self.file_list_label = tk.Label(
            self.expanded_container, 
            text=self._get_text("file_list"), 
            font=("Segoe UI", 10, "bold"), 
            bg=self.bg_color, fg="#888888"
        )
        self.file_list_label.pack(anchor="w", pady=(10, 5))
        
        self.file_table = FileTable(self.expanded_container, height=200)
        self.file_table.pack(fill="x", pady=(0, 10))

        # Hedef klasör seçimi
        self.folder_frame = tk.Frame(self.main_container, bg=self.card_color, bd=1, relief="solid")
        self.folder_frame.pack(fill="x", pady=(0, 10))
        
        folder_inner = tk.Frame(self.folder_frame, bg=self.card_color)
        folder_inner.pack(fill="x", padx=15, pady=10)
        
        tk.Label(folder_inner, text="📂", font=("Segoe UI", 14), bg=self.card_color).pack(side="left")
        
        text_frame = tk.Frame(folder_inner, bg=self.card_color)
        text_frame.pack(side="left", padx=10)
        self.target_folder_tag_label = tk.Label(text_frame, text=self._get_text("target_folder_tag"), font=("Segoe UI", 9, "bold"), bg=self.card_color, fg=self.fg_color)
        self.target_folder_tag_label.pack(anchor="w")
        
        self.folder_path_label = tk.Label(text_frame, text=self._get_text("not_selected"), font=("Segoe UI", 8), bg=self.card_color, fg="#888888")
        self.folder_path_label.pack(anchor="w")
        
        self.select_folder_btn = tk.Button(
            folder_inner, text=self._get_text("select"), bg="#E0E0E0", relief="flat", padx=10, 
            command=self._select_folder
        )
        self.select_folder_btn.pack(side="right")
        
        # Supported Formats Info
        self.formats_label = tk.Label(
            self.main_container, 
            text=self._get_text("supports"),
            font=("Segoe UI", 7), 
            bg=self.bg_color, fg="#AAAAAA"
        )
        self.formats_label.pack(pady=(5, 0))

        # Progress Section
        self.progress = ProgressDialog(self.expanded_container, bg=self.bg_color)
        self.progress.pack(fill="x", pady=(0, 10))

        # Buttons
        self.button_frame = tk.Frame(self.expanded_container, bg=self.bg_color)
        self.button_frame.pack(fill="x")
        
        self.start_btn = tk.Button(
            self.button_frame, text=self._get_text("start"), 
            font=("Segoe UI", 10, "bold"), bg="#4CAF50", fg="white", 
            relief="flat", pady=10, command=self._start_organizing
        )
        self.start_btn.pack(side="right", fill="x", expand=True)

        self.clear_btn = tk.Button(
            self.button_frame, text="🗑️", 
            font=("Segoe UI", 10), bg="#E0E0E0", 
            relief="flat", width=5, command=self._clear_list
        )
        self.clear_btn.pack(side="left", padx=(0, 10))

        self.file_count_label = tk.Label(self.button_frame, text=f"0 {self._get_text('files')}", bg=self.bg_color, fg="#888888")
        self.file_count_label.pack(side="left")

        # Initial Language Update
        self._update_language_ui()
        
        # Apply initial theme for all components (especially if starting in dark mode)
        if self.is_dark_mode:
            self._update_theme_widgets()

    def _toggle_theme(self):
        self.is_dark_mode = not self.is_dark_mode
        config_manager.update_setting("appearance_mode", "dark" if self.is_dark_mode else "light")
        
        # Renkleri ve Iconu Güncelle
        self._apply_theme_colors()
        self.theme_btn.configure(text="☀️" if self.is_dark_mode else "🌙", bg=self.card_color, fg=self.fg_color)
        
        # Optimized theme update - only update necessary widgets
        self._update_theme_widgets()
        logger.info(f"Theme changed: {'Dark' if self.is_dark_mode else 'Light'}")

    def _update_theme_widgets(self):
        """Optimized theme update - only updates changed widgets."""
        # Update special components with their own update methods
        if hasattr(self.drop_zone, "update_theme"):
            self.drop_zone.update_theme(
                self.bg_color, 
                self.card_color, 
                self.fg_color, 
                "#2A2A2A" if self.is_dark_mode else "#EEEEEE"
            )
        
        if hasattr(self.file_table, "update_theme"):
            header_bg = "#333333" if self.is_dark_mode else "#EEEEEE"
            self.file_table.update_theme(
                self.bg_color, 
                self.card_color, 
                self.fg_color, 
                header_bg=header_bg
            )
        
        if hasattr(self.progress, "update_theme"):
            self.progress.update_theme(self.bg_color, self.fg_color)
        
        # Update main container
        self.main_container.configure(bg=self.bg_color)
        self.expanded_container.configure(bg=self.bg_color)
        
        # Update buttons and labels
        self.theme_btn.configure(bg=self.card_color, fg=self.fg_color)
        self.lang_btn.configure(bg=self.card_color, fg=self.fg_color)
        self.logo_text_label.configure(bg=self.bg_color, fg=self.accent_color)
        
        # FIX: Update folder frame and ALL its children for dark mode
        self.folder_frame.configure(bg=self.card_color)
        btn_bg = "#333333" if self.is_dark_mode else "#E0E0E0"
        btn_fg = "#FFFFFF" if self.is_dark_mode else "#333333"
        self.select_folder_btn.configure(bg=btn_bg, fg=btn_fg)
        self.target_folder_tag_label.configure(bg=self.card_color, fg=self.fg_color)
        self.folder_path_label.configure(bg=self.card_color, fg="#888888")
        self.formats_label.configure(bg=self.bg_color)
        self.clear_btn.configure(bg=btn_bg, fg=btn_fg)
        self.button_frame.configure(bg=self.bg_color)
        
        # Recursively update folder_frame children
        for child in self.folder_frame.winfo_children():
            try:
                child.configure(bg=self.card_color)
                for subchild in child.winfo_children():
                    try:
                        if isinstance(subchild, tk.Label):
                            subchild.configure(bg=self.card_color)
                        elif isinstance(subchild, tk.Frame):
                            subchild.configure(bg=self.card_color)
                    except Exception:
                        pass
            except Exception:
                pass
        
        # Update other labels
        for widget in [self.file_list_label, self.file_count_label]:
            widget.configure(bg=self.bg_color, fg="#888888")

    def _get_text(self, key, **kwargs):
        """Helper to get translated text."""
        return get_text(self.lang, key, **kwargs)

    def _toggle_language(self):
        """Switch between TR and EN."""
        self.lang = "tr" if self.lang == "en" else "en"
        config_manager.update_setting("language", self.lang)
        self._update_language_ui()
        logger.info(f"Language changed to: {self.lang}")

    def _update_language_ui(self):
        """Refreshes all UI text elements with current language."""
        self.lang_btn.configure(text=self._get_text("lang_name"))
        self.file_list_label.configure(text=self._get_text("file_list"))
        self.target_folder_tag_label.configure(text=self._get_text("target_folder_tag"))
        
        if not self.target_folder:
            self.folder_path_label.configure(text=self._get_text("not_selected"))
        
        self.select_folder_btn.configure(text=self._get_text("select"))
        self.formats_label.configure(text=self._get_text("supports"))
        self.start_btn.configure(text=self._get_text("start"))
        self.file_count_label.configure(text=f"{len(self.files_data)} {self._get_text('files')}")
        
        # Components
        self.drop_zone.update_labels(self._get_text("drop_main"), self._get_text("drop_sub"))
        self.file_table.refresh_headers([
            self._get_text("header_file"), 
            self._get_text("header_date"), 
            self._get_text("header_device"), 
            self._get_text("header_path")
        ])
        self.progress.update_labels(self._get_text("ready"))

    def _setup_dnd(self):
        self.drop_zone.drop_target_register(DND_FILES)
        self.drop_zone.dnd_bind('<<Drop>>', self._on_drop)

    def _on_drop(self, event):
        self.drop_zone.highlight(False)
        
        # Security: Check file limit before processing
        if len(self.files_data) >= MAX_FILES_LIMIT:
            messagebox.showwarning("Lume", self._get_text("warn_file_limit"))
            return
        
        # Zen modundaysa ve bir şey bırakıldıysa hemen genişlet
        if self.is_zen_mode: 
            self._expand_ui()
        
        paths = self._parse_drop_data(event.data)
        logger.info(f"Dropped {len(paths)} items")
        
        added_count = 0
        unsupported_formats = False
        duplicates = False
        security_blocked = False
        
        for path in paths:
            # Temizlik: Süslü parantez, tırnak ve gizli boşlukları temizle
            path = path.strip('{}').strip('"').strip("'").strip()
            if not path: 
                continue
            
            # Security: Path length check
            if len(path) > MAX_PATH_LENGTH:
                logger.warning(f"Path too long (>{MAX_PATH_LENGTH} chars), skipped")
                security_blocked = True
                continue
            
            # Windows dosya yolu düzeltmesi
            path = os.path.normpath(path)
            
            # Security: Validate path before processing
            if not self._is_safe_path(path):
                security_blocked = True
                continue
            
            # Security: Check file limit
            if len(self.files_data) >= MAX_FILES_LIMIT:
                messagebox.showwarning("Lume", self._get_text("warn_file_limit"))
                break
            
            if os.path.isfile(path):
                if is_supported_image(path):
                    result = self._add_file(path)
                    if result == "added":
                        added_count += 1
                    elif result == "duplicate":
                        duplicates = True
                    elif result == "blocked":
                        security_blocked = True
                else:
                    unsupported_formats = True
            elif os.path.isdir(path):
                for root, _, files in os.walk(path):
                    for f in files:
                        if len(self.files_data) >= MAX_FILES_LIMIT:
                            break
                        
                        fp = os.path.join(root, f)
                        
                        # Security: Skip if path too long
                        if len(fp) > MAX_PATH_LENGTH:
                            continue
                        
                        if is_supported_image(fp):
                            result = self._add_file(fp)
                            if result == "added":
                                added_count += 1
                            elif result == "duplicate":
                                duplicates = True
                            elif result == "blocked":
                                security_blocked = True
                        else:
                            unsupported_formats = True
        
        self.file_count_label.configure(text=f"{len(self.files_data)} {self._get_text('files')}")
        
        # Status messages
        if security_blocked:
            self._show_status(self._get_text("err_security"), "red")
        elif added_count == 0:
            if unsupported_formats:
                self._show_status(self._get_text("err_unsupported"), "red")
            elif duplicates:
                self._show_status(self._get_text("err_duplicates"), "orange")
            else:
                self._show_status(self._get_text("err_none"), "orange")
        elif unsupported_formats or duplicates:
            self._show_status(self._get_text("success_added", count=added_count), "blue")

    def _is_safe_path(self, path):
        """Security: Validates path for safety."""
        try:
            # Resolve real path (detects symlinks, junctions, etc.)
            real_path = os.path.realpath(path)
            
            # Check if path was modified (symlink/junction detection)
            norm_path = os.path.normpath(path)
            if real_path != norm_path and os.path.islink(path):
                logger.warning(f"Symlink/junction blocked: {os.path.basename(path)}")
                return False
            
            # Block .lnk files
            if path.lower().endswith('.lnk'):
                logger.warning(f"Shortcut file blocked: {os.path.basename(path)}")
                return False
            
            # Path traversal check
            if '..' in path.split(os.sep):
                logger.warning(f"Path traversal attempt blocked: {path}")
                return False
            
            # Block system directories
            system_paths = [
                os.environ.get('SYSTEMROOT', 'C:\\Windows').lower(),
                os.environ.get('PROGRAMFILES', 'C:\\Program Files').lower(),
            ]
            
            real_path_lower = real_path.lower()
            for sys_path in system_paths:
                if real_path_lower.startswith(sys_path):
                    logger.warning(f"System path blocked: {path}")
                    return False
            
            return True
            
        except Exception as e:
            logger.error(f"Path validation failed: {str(e)}")
            return False

    def _show_status(self, text, color=None):
        """Shows a temporary status message at the bottom."""
        if not color: 
            color = self.fg_color
        
        # Save current state
        old_text = f"{len(self.files_data)} {self._get_text('files')}"
        old_color = "#888888"
        
        # Show new message
        self.file_count_label.configure(text=text, fg=color)
        logger.info(f"Status: {text}")
        
        # Reset after 4 seconds
        self.after(4000, lambda: self.file_count_label.configure(text=old_text, fg=old_color))

    def _parse_drop_data(self, data):
        """Tcl list parçalama."""
        return self.tk.splitlist(data)

    def _expand_ui(self):
        if self.is_zen_mode:
            self.is_zen_mode = False
            self.resizable(True, True)
            self.geometry("800x800")
            self.expanded_container.pack(fill="both", expand=True)

    def _add_file(self, file_path):
        """Adds file to list with security checks. Returns: 'added', 'duplicate', 'blocked'"""
        if self.is_zen_mode: 
            self._expand_ui()
        
        # Security: Additional path validation
        if not self._is_safe_path(file_path):
            return "blocked"
        
        # Optimized: Calculate hash once
        file_hash = get_file_hash(file_path, quick=True)
        if not file_hash:
            logger.warning(f"Hash calculation failed: {os.path.basename(file_path)}")
            return "blocked"
        
        if file_hash in self.added_hashes: 
            return "duplicate"
        
        info = get_file_info(file_path)
        if not info:  # Rejection (Symlink etc)
            return "blocked"
        
        # Store both hashes
        info['quick_hash'] = file_hash
        info['full_hash'] = get_file_hash(file_path, quick=False)
        
        if self.target_folder:
            new_path = calculate_new_path(info, self.target_folder)
            rel = get_relative_path(new_path, self.target_folder)
        else:
            rel = "..."
        
        self.files_data.append(info)
        self.added_hashes.add(file_hash)
        
        self.file_table.add_row(
            info['filename'], 
            info['date_str'], 
            info['device'], 
            self._sanitize_path_display(rel, 30)
        )
        return "added"

    def _select_folder(self):
        folder = filedialog.askdirectory(title="Select Target Folder")
        if folder:
            # Security: Validate folder
            if not self._validate_target_folder(folder):
                messagebox.showerror("Lume", self._get_text("err_invalid_folder"))
                return
            
            self.target_folder = folder
            config_manager.update_setting("target_folder", folder)
            display_path = self._sanitize_path_display(folder, 40)
            self.folder_path_label.configure(text=display_path)
            logger.info("Target folder selected: ***")

    def _clear_list(self):
        """Clears the current file list."""
        self.files_data = []
        self.added_hashes = set()
        self.file_table.clear()
        self.progress.reset()
        self.progress.update_labels(self._get_text("ready"))
        self.file_count_label.configure(text=f"0 {self._get_text('files')}")

    def _start_organizing(self):
        """Starts the organization process."""
        if not self.target_folder:
            messagebox.showwarning("Lume", self._get_text("warn_select_folder"))
            return
        
        if not self.files_data:
            messagebox.showwarning("Lume", self._get_text("warn_no_files"))
            return
        
        # Security: Re-validate target folder before starting
        if not self._validate_target_folder(self.target_folder):
            messagebox.showerror("Lume", self._get_text("err_invalid_folder"))
            return
        
        self.start_btn.configure(text=self._get_text("status_organizing"), state="disabled")
        self.progress.reset()
        
        threading.Thread(target=self._organize_files_thread, daemon=True).start()

    def _organize_files_thread(self):
        total = len(self.files_data)
        success = 0
        
        for i, info in enumerate(self.files_data):
            try:
                # Rate limiting to prevent CPU overload
                time.sleep(0.01)
                
                ok = move_file(info, self.target_folder)
                if ok: 
                    success += 1
            except Exception as e:
                # Security: Log details, show generic to user
                logger.error(f"Error moving file: {str(e)}", exc_info=True)
            
            # Update progress
            self.after(0, lambda c=i+1: self.progress.update_progress(
                c, total, 
                self._get_text("processing", percentage=int((c/total)*100), current=c, total=total)
            ))
        
        self.after(0, lambda: self._on_complete(success))

    def _on_complete(self, count):
        self.progress.complete(count, self._get_text("status_archived"))
        self.start_btn.configure(text=self._get_text("start"), state="normal")
        messagebox.showinfo("Lume", self._get_text("info_complete", count=count))


if __name__ == "__main__":
    app = LumeApp()
    app.mainloop()

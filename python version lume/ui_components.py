import tkinter as tk
from tkinter import ttk

class DropZone(tk.Frame):

    def __init__(self, master, drop_callback=None, **kwargs):
        self.bg_color = kwargs.pop('bg_color', "#F5F5F5")
        self.active_color = kwargs.pop('active_color', "#E0E0E0")

        super().__init__(master, bg=self.bg_color, bd=2, relief="groove", **kwargs)

        self.drop_callback = drop_callback

        self.inner_label = tk.Label(
            self,
            text="",
            font=("Segoe UI", 14, "bold"),
            bg=self.bg_color,
            fg="#555555"
        )
        self.inner_label.pack(fill="x", pady=(35, 5))

        self.subtext = tk.Label(
            self,
            text="",
            font=("Segoe UI", 10),
            bg=self.bg_color,
            fg="#666666"
        )
        self.subtext.pack(fill="x", pady=(0, 35))

    def update_labels(self, main_text, sub_text):

        self.inner_label.configure(text=main_text)
        self.subtext.configure(text=sub_text)

    def highlight(self, active=True):

        color = self.active_color if active else self.bg_color
        self.configure(bg=color)
        self.inner_label.configure(bg=color)
        self.subtext.configure(bg=color)

    def update_theme(self, bg_color, card_color, fg_color, active_color):

        self.bg_color = card_color
        self.active_color = active_color
        self.configure(bg=self.bg_color)
        self.inner_label.configure(bg=self.bg_color, fg=fg_color)
        self.subtext.configure(bg=self.bg_color)

class FileTable(tk.Frame):

    def __init__(self, master, height=200, **kwargs):

        self.current_bg = "#FFFFFF"
        self.current_fg = "#333333"
        self.current_header_bg = "#EEEEEE"
        self.is_dark = False

        kwargs['bg'] = self.current_bg
        super().__init__(master, **kwargs)

        self.canvas = tk.Canvas(self, height=height, bg=self.current_bg, highlightthickness=0)
        self.scrollable_frame = tk.Frame(self.canvas, bg=self.current_bg)

        self.scrollable_frame.bind(
            "<Configure>",
            lambda e: self.canvas.configure(scrollregion=self.canvas.bbox("all"))
        )

        self.canvas_window = self.canvas.create_window((0, 0), window=self.scrollable_frame, anchor="nw")
        self.bind("<Configure>", self._on_resize)

        self.canvas.bind("<MouseWheel>", self._on_mousewheel)
        self.canvas.bind("<Enter>", lambda e: self.canvas.bind_all("<MouseWheel>", self._on_mousewheel))
        self.canvas.bind("<Leave>", lambda e: self.canvas.unbind_all("<MouseWheel>"))

        self.canvas.pack(fill="both", expand=True)

        self.rows = []
        self._create_header()

    def _on_mousewheel(self, event):
        self.canvas.yview_scroll(int(-1*(event.delta/120)), "units")

    def _on_resize(self, event):
        self.canvas.itemconfig(self.canvas_window, width=event.width)

    def _create_header(self, col_names=None):
        if hasattr(self, 'header_frame'):
            self.header_frame.destroy()

        self.header_frame = tk.Frame(self.scrollable_frame, bg=self.current_header_bg)
        self.header_frame.pack(fill="x", pady=(0, 5), before=self.scrollable_frame.winfo_children()[0] if self.rows else None)

        if not col_names:
            col_names = ["File", "Date", "Device", "New Path"]

        columns = [(col_names[0], 0.3), (col_names[1], 0.2), (col_names[2], 0.2), (col_names[3], 0.3)]
        for text, weight in columns:
            lbl = tk.Label(self.header_frame, text=text, font=("Segoe UI", 9, "bold"), bg=self.current_header_bg, fg=self.current_fg, anchor="w", padx=10)
            lbl.pack(side="left", expand=True, fill="x")

    def refresh_headers(self, col_names):

        self._create_header(col_names)

    def add_row(self, filename, date, device, new_path):
        row = tk.Frame(self.scrollable_frame, bg=self.current_bg)
        row.pack(fill="x", pady=1)

        vals = [filename, date, device, new_path]
        for val in vals:
            display = val if len(val) <= 30 else "..." + val[-27:]
            lbl = tk.Label(row, text=display, font=("Segoe UI", 9), bg=self.current_bg, fg=self.current_fg, anchor="w", padx=10)
            lbl.pack(side="left", expand=True, fill="x")

        self.rows.append(row)

    def clear(self):
        for row in self.rows:
            row.destroy()
        self.rows = []

    def update_theme(self, bg_color, card_color, fg_color, header_bg=None):

        self.current_bg = card_color
        self.current_fg = fg_color
        self.is_dark = fg_color == "#FFFFFF"
        self.current_header_bg = header_bg if header_bg else ("#333333" if self.is_dark else "#EEEEEE")

        self.configure(bg=card_color)
        self.canvas.configure(bg=card_color)
        self.scrollable_frame.configure(bg=card_color)

        for child in self.scrollable_frame.winfo_children():
            if isinstance(child, tk.Frame):
                is_header = child == getattr(self, 'header_frame', None)
                new_bg = self.current_header_bg if is_header else card_color
                child.configure(bg=new_bg)

                for label in child.winfo_children():
                    if isinstance(label, tk.Label):
                        label.configure(bg=new_bg, fg=fg_color)

class ProgressDialog(tk.Frame):

    def __init__(self, master, **kwargs):

        self.current_bg = kwargs.get('bg', '#FFFFFF')
        self.is_dark = False
        super().__init__(master, **kwargs)

        self.progress_var = tk.DoubleVar()
        self.bar_height = 8

        self.trough_color = "#E0E0E0"
        self.bar_color = "#4CAF50"

        self.progress_canvas = tk.Canvas(
            self,
            height=self.bar_height,
            bg=self.trough_color,
            highlightthickness=0
        )
        self.progress_canvas.pack(pady=10, padx=20, fill="x")

        self.fill_rect = self.progress_canvas.create_rectangle(
            0, 0, 0, self.bar_height,
            fill=self.bar_color,
            outline=""
        )

        self.status_label = tk.Label(
            self,
            text="",
            font=("Segoe UI", 9),
            bg=self.current_bg,
            fg="#666666"
        )
        self.status_label.pack()

    def update_progress(self, current, total, message=None):
        if total > 0:
            percentage = (current / total) * 100
            self.progress_var.set(percentage)

            canvas_width = self.progress_canvas.winfo_width()
            if canvas_width > 1:
                fill_width = int((percentage / 100) * canvas_width)
                self.progress_canvas.coords(self.fill_rect, 0, 0, fill_width, self.bar_height)

            status = message if message else f"{int(percentage)}% - {current}/{total} files processed"
            self.status_label.configure(text=status)

    def complete(self, success_count, template):

        self.progress_var.set(100)

        canvas_width = self.progress_canvas.winfo_width()
        self.progress_canvas.coords(self.fill_rect, 0, 0, canvas_width, self.bar_height)
        self.progress_canvas.itemconfig(self.fill_rect, fill="#4CAF50")

        self.status_label.configure(
            text=template.format(count=success_count),
            fg="green"
        )

    def update_labels(self, ready_text):

        current_text = self.status_label.cget("text")
        if "archived" not in current_text and "arşivlendi" not in current_text and "✅" not in current_text and "%" not in current_text:
            self.status_label.configure(text=ready_text)

    def reset(self):

        self.progress_var.set(0)
        self.progress_canvas.coords(self.fill_rect, 0, 0, 0, self.bar_height)
        self.progress_canvas.itemconfig(self.fill_rect, fill=self.bar_color)
        self.status_label.configure(text="", fg="#AAAAAA" if self.is_dark else "#666666")

    def update_theme(self, bg_color, fg_color):
        self.current_bg = bg_color
        self.is_dark = fg_color == "#FFFFFF"
        self.configure(bg=bg_color)
        self.status_label.configure(bg=bg_color)

        self.trough_color = "#333333" if self.is_dark else "#E0E0E0"
        self.bar_color = "#6366f1" if self.is_dark else "#4CAF50"

        self.progress_canvas.configure(bg=self.trough_color)

        current_fill = self.progress_canvas.itemcget(self.fill_rect, "fill")
        if current_fill != "#4CAF50":
            self.progress_canvas.itemconfig(self.fill_rect, fill=self.bar_color)

        current_fg = self.status_label.cget("fg")
        if current_fg != "green":
            new_fg = "#AAAAAA" if self.is_dark else "#666666"
            self.status_label.configure(fg=new_fg)

import builtins
import io
import os
import shutil
from typing import Any, Optional

from PIL import Image as PilImage
from kivy.core.image import Image as CoreImage
from kivy.metrics import dp
from kivy.uix.image import Image
from kivy.uix.screenmanager import Screen
from kivymd.app import MDApp
from kivymd.uix.button import MDFlatButton
from kivymd.uix.dialog import MDDialog


class PreviewScreen(Screen):
    """Kivy screen that displays a scrollable preview grid and handles discarding/keeping images."""

    def __init__(self, **kwargs: Any) -> None:
        super().__init__(**kwargs)
        self.attachment_dir: Optional[str] = None
        self.dialog: Optional[MDDialog] = None

    @property
    def config(self):
        """Return the application configuration."""
        return MDApp.get_running_app().app_config

    def set_attachment_dir(self, attachment_dir: str) -> None:
        """Set the directory containing images to preview."""
        self.attachment_dir = attachment_dir

    def on_enter(self) -> None:
        """Load images into the grid layout using thumbnails."""
        grid = self.ids.preview_grid
        grid.clear_widgets()

        if not self.attachment_dir or not os.path.exists(self.attachment_dir):
            return

        files = sorted([f for f in os.listdir(self.attachment_dir) if f.lower().endswith(".jpg")])

        for filename in files:
            file_path = os.path.join(self.attachment_dir, filename)

            try:
                pil_img = PilImage.open(file_path)

                # Resize to a thumbnail size to speed up loading and improve visual quality (antialiasing)
                pil_img.thumbnail((640, 480), PilImage.Resampling.LANCZOS)

                data = io.BytesIO()
                pil_img.save(data, format='jpeg', quality=80)
                data.seek(0)

                im_data = CoreImage(data, ext='jpeg')

                img_widget = Image(
                    texture=im_data.texture,
                    size_hint=(1, None),
                    height=dp(240),
                    fit_mode="contain",
                    nocache=True
                )
                grid.add_widget(img_widget)

            except Exception as e:
                print(f"Error loading preview image {filename}: {e}")

    def on_discard_pressed(self) -> None:
        """Shows a confirmation dialog to discard images."""
        if self.dialog:
            self.dialog.dismiss()

        self.dialog = MDDialog(
            title=builtins._("preview_dialog_title"),
            text=builtins._("preview_dialog_text"),
            buttons=[
                MDFlatButton(text=builtins._("preview_dialog_back"), on_press=self._dismiss_dialog),
                MDFlatButton(text=builtins._("preview_dialog_confirm"), on_press=self._confirm_discard)
            ]
        )
        self.dialog.open()

    def on_keep_pressed(self) -> None:
        """Proceed to the share screen."""
        share_screen = self.manager.get_screen("share_screen")
        share_screen.set_attachment_dir(self.attachment_dir)
        self.manager.current = "share_screen"

    def _dismiss_dialog(self, instance: Any) -> None:
        """Close the dialog without taking action."""
        if self.dialog:
            self.dialog.dismiss()

    def _confirm_discard(self, instance: Any) -> None:
        """Move the entire session folder to a 'Trash' subdirectory and return to the welcome screen."""
        if self.dialog:
            self.dialog.dismiss()

        if self.attachment_dir and os.path.exists(self.attachment_dir):
            try:
                trash_root = self.config.images.base_image_dir / "Trash"
                trash_root.mkdir(parents=True, exist_ok=True)

                folder_name = os.path.basename(os.path.normpath(self.attachment_dir))
                target_path = trash_root / folder_name

                if target_path.exists():
                    shutil.rmtree(target_path)

                shutil.move(self.attachment_dir, str(target_path))
                print(f"Moved session {folder_name} to Trash: {target_path}")

            except Exception as e:
                print(f"Failed to move images to trash: {e}")

        self.manager.current = "welcome_screen"

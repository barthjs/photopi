import builtins
import os
from datetime import datetime
from typing import Any, Optional

from PIL import Image as PilImage
from kivy.animation import Animation
from kivy.clock import Clock
from kivy.uix.screenmanager import Screen
from kivymd.app import MDApp

from photopi.camera.live_preview import LivePreview


# noinspection PyProtectedMember,PyUnresolvedReferences,PyUnusedLocal
class LiveViewScreen(Screen):
    """Kivy screen handling live camera preview and image capturing."""

    def __init__(self, **kwargs: Any) -> None:
        super().__init__(**kwargs)
        self.inactive: bool = True
        self.current_capture_dir: Optional[str] = None
        self.countdown: Optional[int] = None
        self.image_count: int = 0

    @property
    def config(self):
        """Return the application configuration."""
        return MDApp.get_running_app().app_config

    def on_enter(self) -> None:
        """Called when the screen is entered; start live preview updates."""
        self.inactive = True
        Clock.schedule_once(self._reset_live_preview, 0)
        Clock.schedule_once(self._check_activity, 60)
        Clock.schedule_interval(self.ids.live_preview.update_frame, 1.0 / 30.0)

    def on_leave(self) -> None:
        """Called when the screen is left; stop live preview updates."""
        Clock.schedule_once(self._reset_live_preview, 0)
        Clock.unschedule(self.ids.live_preview.update_frame)

    def start_sequence(self, instance: Any) -> None:
        """Start the image capture sequence."""
        self.inactive = False
        self._create_image_dir()
        self.image_count = 0
        self.ids.capture_button.opacity = 0
        Clock.schedule_once(self._start_countdown, 1)

    def _reset_live_preview(self, dt: float) -> None:
        """Reset the live preview display."""
        self.ids.live_preview.update_frame(dt)
        self.ids.progress_label.text = ""
        self.ids.countdown_label.text = ""
        self.ids.capture_button.opacity = 1

    def _check_activity(self, dt: float) -> None:
        """Return to the welcome screen if no activity for a while."""
        if self.inactive:
            Clock.schedule_once(self._return_to_welcome_screen, 1)

    def _return_to_welcome_screen(self, dt: float) -> None:
        """Navigate to the welcome screen."""
        self.manager.current = "welcome_screen"

    def _create_image_dir(self) -> None:
        """Create a new directory for storing captured images."""
        prefix = self.config.images.file_prefix
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")

        # base_image_dir / file_prefix / file_prefix_datetime
        parent_dir = self.config.images.base_image_dir / prefix
        dir_name = f"{prefix}_{timestamp}"
        full_path = parent_dir / dir_name

        try:
            full_path.mkdir(parents=True, exist_ok=True)
            self.current_capture_dir = str(full_path)
        except PermissionError as e:
            raise PermissionError(
                f"Permission denied: Cannot create image directory '{full_path}'. "
                "Check directory permissions."
            ) from e
        except OSError as e:
            raise OSError(
                f"Failed to create image directory '{full_path}': {e.strerror}"
            ) from e

    def _start_countdown(self, dt: float) -> None:
        """Start the countdown before capturing an image."""
        self.countdown = 10
        self.ids.countdown_label.opacity = 1
        self.ids.countdown_label.text = str(self.countdown)
        Clock.schedule_interval(self._update_countdown, 1)

    def _update_countdown(self, dt: float) -> None:
        """Update the countdown display and trigger image capture when it reaches zero."""
        if self.countdown is None:
            return

        self.countdown -= 1
        self.ids.countdown_label.text = str(self.countdown)

        if self.countdown == 0:
            Clock.unschedule(self._update_countdown)
            self.ids.countdown_label.opacity = 0
            self._capture_image()
            self.image_count += 1
            self.ids.progress_label.text = builtins._("Images: {}/{}").format(
                self.image_count, self.config.images.max_image_count
            )

            if self.image_count >= self.config.images.max_image_count:
                self._end_sequence()
            else:
                Clock.schedule_once(self._start_countdown, 5)

    def _capture_image(self) -> None:
        """Capture an image from the camera and save it with an optional overlay."""
        self._trigger_flash()
        live_preview = self.ids.live_preview
        with LivePreview._lock:
            live_preview.cam.switch_mode(live_preview.capture_config)
            frame = live_preview.cam.capture_array()
            live_preview.cam.switch_mode(live_preview.preview_config)

        if frame is None:
            raise RuntimeError("Failed to capture image from camera.")

        if self.current_capture_dir is None:
            raise ValueError("Capture directory is not set for saving images.")

        timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
        filename = os.path.join(self.current_capture_dir,
                                f"{self.config.images.file_prefix}-{timestamp}_{self.image_count}.jpg")

        try:
            pil_image = PilImage.fromarray(frame).transpose(PilImage.FLIP_TOP_BOTTOM)
            self._apply_overlay(pil_image)
            pil_image.save(filename)
        except PermissionError as e:
            raise PermissionError(
                f"Permission denied: Cannot save image '{filename}'. "
                "Check file system permissions and available disk space."
            ) from e
        except OSError as e:
            raise OSError(
                f"Failed to save image '{filename}': {e.strerror}"
            ) from e
        except Exception as e:
            raise RuntimeError(f"Unexpected error while saving image '{filename}': {e}") from e

    def _trigger_flash(self) -> None:
        """Trigger a brief white flash animation."""
        flash = self.ids.flash_widget
        flash.opacity = 1
        anim = Animation(opacity=0, duration=0.5)
        anim.start(flash)

    def _apply_overlay(self, pil_image: PilImage.Image) -> None:
        """Apply an overlay to the image if available."""
        if not self.config.images.final_overlay:
            return

        try:
            overlay = PilImage.open(self.config.images.final_overlay).resize((4000, 2400)).convert("RGBA")
            pil_image.paste(overlay, (0, 0), overlay)
        except Exception as e:
            print(f"Warning: Could not load overlay. Error: {e}")

    def _end_sequence(self) -> None:
        """Show a completion message and schedule transition to the preview screen."""
        self.ids.countdown_label.opacity = 1
        self.ids.countdown_label.text = builtins._("Capture Complete!")
        Clock.schedule_once(self._show_preview_screen, 4)

    def _show_preview_screen(self, dt: float) -> None:
        """Navigate to the preview screen."""
        preview_screen = self.manager.get_screen("preview_screen")
        preview_screen.set_attachment_dir(self.current_capture_dir)
        self.manager.current = "preview_screen"

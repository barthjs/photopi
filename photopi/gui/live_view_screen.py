import builtins
import os
from datetime import datetime
from typing import Any, Dict, Optional

from PIL import Image as PilImage
from kivy.clock import Clock
from kivy.uix.screenmanager import Screen

from photopi.camera.live_preview import LivePreview


# noinspection PyProtectedMember,PyUnresolvedReferences,PyUnusedLocal
class LiveViewScreen(Screen):
    """Kivy screen handling live camera preview and image capture sequence."""

    def __init__(self, config: Dict[str, Any], **kwargs: Any) -> None:
        super().__init__(**kwargs)
        self.inactive: bool = True
        self.dir_index: Optional[int] = None
        self.countdown: Optional[int] = None
        self.image_count: int = 0
        images_config = config["images"]
        self.base_image_dir = images_config.get("base_image_dir")
        self.max_image_count = images_config["max_image_count"]
        self.overlay_path = images_config.get("final_overlay")

    def start_sequence(self, instance: Any) -> None:
        """Start the image capture sequence."""
        self.inactive = False
        self._set_next_dir_index()
        self._create_image_dir()
        self.image_count = 0
        self.ids.capture_button.opacity = 0
        Clock.schedule_once(self._start_countdown, 1)

    def _set_next_dir_index(self) -> None:
        """Find the next available directory index starting from 0000."""
        dir_index = 0
        while os.path.exists(os.path.join(self.base_image_dir, f"{dir_index:04d}")):
            dir_index += 1
        self.dir_index = dir_index

    def _create_image_dir(self) -> None:
        """Create a new directory for storing captured images."""
        if self.dir_index is None:
            raise ValueError("Directory index must be set before creating image directory.")

        dir_name = os.path.join(self.base_image_dir, f"{self.dir_index:04d}")
        try:
            os.makedirs(dir_name, exist_ok=False)
        except FileExistsError:
            # Directory exists, possibly from a previous run
            pass
        except PermissionError as e:
            raise PermissionError(
                f"Permission denied: Cannot create image directory '{dir_name}'. "
                "Check directory permissions."
            ) from e
        except OSError as e:
            raise OSError(
                f"Failed to create image directory '{dir_name}': {e.strerror}"
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
                self.image_count, self.max_image_count
            )

            if self.image_count >= self.max_image_count:
                self._end_sequence()
            else:
                Clock.schedule_once(self._start_countdown, 5)

    def _capture_image(self) -> None:
        """Capture an image from the camera and save it with an optional overlay."""
        live_preview = self.ids.live_preview
        with LivePreview._lock:
            live_preview.cam.switch_mode(live_preview.capture_config)
            frame = live_preview.cam.capture_array()
            live_preview.cam.switch_mode(live_preview.preview_config)

        if frame is None:
            raise RuntimeError("Failed to capture image from camera.")

        if self.dir_index is None:
            raise ValueError("Directory index is not set for saving images.")

        dir_path = os.path.join(self.base_image_dir, f"{self.dir_index:04d}")
        timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
        filename = os.path.join(dir_path, f"PhotoPi-{timestamp}_{self.image_count}.jpg")

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

    def _apply_overlay(self, pil_image: PilImage.Image) -> None:
        """Apply an overlay to the image if available."""
        if not self.overlay_path:
            return

        try:
            overlay = PilImage.open(self.overlay_path).resize((4000, 2400)).convert("RGBA")
            pil_image.paste(overlay, (0, 0), overlay)
        except Exception as e:
            print(f"Warning: Could not load overlay. Error: {e}")

    def _end_sequence(self) -> None:
        """Show a completion message and schedule transition to the email screen."""
        self.ids.countdown_label.opacity = 1
        self.ids.countdown_label.text = builtins._("Capture Complete!")
        Clock.schedule_once(self._return_to_email_screen, 4)

    def _return_to_email_screen(self, dt: float) -> None:
        """Navigate to the email screen with an attachment directory set."""
        if self.dir_index is None:
            return

        email_screen = self.manager.get_screen("email_screen")
        attachment_dir = os.path.join(self.base_image_dir, f"{self.dir_index:04d}")
        email_screen.set_attachment_dir(attachment_dir)
        self.manager.current = "email_screen"

    def _return_to_welcome_screen(self, dt: float) -> None:
        """Navigate to the welcome screen."""
        self.manager.current = "welcome_screen"

    def _reset_live_preview(self, dt: float) -> None:
        """Reset the live preview display."""
        self.ids.live_preview.update_frame(dt)
        self.ids.progress_label.text = ""
        self.ids.countdown_label.text = ""
        self.ids.capture_button.opacity = 1

    def _check_activity(self, dt: float) -> None:
        """Return to welcome screen if screen is inactive."""
        if self.inactive:
            Clock.schedule_once(self._return_to_welcome_screen, 1)

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

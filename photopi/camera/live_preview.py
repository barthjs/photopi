from typing import Optional, Any

import numpy as np
from PIL import Image as PILImage
from kivy.graphics.texture import Texture
from kivy.properties import StringProperty
from kivy.uix.image import Image
from libcamera import Transform
from picamera2 import Picamera2


class LivePreview(Image):
    """
    Widget displaying live camera preview with an optional overlay image.

    Manages the camera preview and still capture configurations, captures frames,
    applies overlay if set, and updates the displayed texture.
    """
    overlay_path: StringProperty = StringProperty("")

    def __init__(self, **kwargs) -> None:
        super().__init__(**kwargs)
        self.overlay_image: Optional[PILImage.Image] = None
        self.cam = Picamera2()
        self.cam.options["quality"] = 95

        self.preview_config = self.cam.create_preview_configuration(
            main={"size": (800, 480), "format": "BGR888"},
            transform=Transform(hflip=0, vflip=1)
        )

        self.capture_config = self.cam.create_still_configuration(
            main={"size": (4000, 2400), "format": "BGR888"},
            transform=Transform(hflip=0, vflip=1)
        )

        self.cam.configure(self.preview_config)
        self.cam.start()

    def on_overlay_path(self, instance: Any, value: str) -> None:
        """
        Called automatically when the 'overlay_path' property changes.
        Triggered by Kivy when the property is set (from the .kv file).
        Loads and prepares the overlay image based on the path.
        """
        self.overlay_image = None
        if not value:
            return

        try:
            overlay = PILImage.open(value)
            overlay = overlay.resize((800, 480)).transpose(PILImage.FLIP_TOP_BOTTOM).convert("RGBA")
            self.overlay_image = overlay
        except Exception as e:
            print(f"Error loading overlay image '{value}': {e}")

    def update_frame(self, dt: float) -> None:
        """Capture a frame from the camera, apply overlay if available, and update texture."""
        frame = self.cam.capture_array()

        if self.overlay_image:
            frame_image = PILImage.fromarray(frame)
            frame_image.paste(self.overlay_image, (0, 0), self.overlay_image)
            frame = np.array(frame_image)

        # Convert the frame to a texture
        texture = Texture.create(size=(frame.shape[1], frame.shape[0]))
        texture.blit_buffer(frame.tobytes())
        self.texture = texture

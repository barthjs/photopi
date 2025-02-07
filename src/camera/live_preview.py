import numpy as np
from PIL import Image as PILImage
from kivy.graphics.texture import Texture
from kivy.properties import StringProperty
from kivy.uix.image import Image
from libcamera import Transform
from picamera2 import Picamera2


class LivePreview(Image):
    overlay_path = StringProperty("")

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.overlay_image = None
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

    def on_overlay_path(self, instance, value):
        """Loads the overlay image based on the specified overlay_path."""
        try:
            self.overlay_image = PILImage.open(self.overlay_path).resize((800, 480)).transpose(
                PILImage.FLIP_TOP_BOTTOM).convert("RGBA")
        except Exception as e:
            print(f"Error loading overlay: {e}")

    def update_frame(self, dt):
        """Captures a frame, applies overlay, and updates the displayed texture."""
        frame = self.cam.capture_array()

        if self.overlay_image:
            frame_image = PILImage.fromarray(frame)
            frame_image.paste(self.overlay_image, (0, 0), self.overlay_image)
            frame = np.array(frame_image)

        # Convert the frame to a texture
        texture = Texture.create(size=(frame.shape[1], frame.shape[0]))
        texture.blit_buffer(frame.tobytes())
        self.texture = texture

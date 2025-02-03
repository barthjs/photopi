import numpy as np
from PIL import Image as PILImage
from kivy.graphics.texture import Texture
from kivy.uix.image import Image
from libcamera import Transform
from picamera2 import Picamera2


class LivePreview(Image):
    def __init__(self, overlay_path="", **kwargs):
        super().__init__(**kwargs)
        self.overlay_image = None
        self.cam = Picamera2()
        self.cam.options["quality"] = 95

        if overlay_path:
            self.overlay_path = overlay_path
            self.overlay_image = PILImage.open(self.overlay_path).resize((800, 480)).transpose(
                PILImage.FLIP_TOP_BOTTOM).convert("RGBA")

        self.preview_config = self.cam.create_preview_configuration(
            main={"size": (800, 480), "format": "BGR888"},
            transform=Transform(hflip=0, vflip=1)
        )

        self.capture_config = self.cam.create_still_configuration(
            main={"format": "BGR888"},
            transform=Transform(hflip=0, vflip=1)
        )

        self.cam.configure(self.preview_config)
        self.cam.start()

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

    def stop(self):
        # Stop the camera
        self.cam.stop()

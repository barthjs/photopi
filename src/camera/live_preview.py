from kivy.graphics.texture import Texture
from kivy.uix.image import Image
from libcamera import Transform
from picamera2 import Picamera2


class LivePreview(Image):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.cam = Picamera2()
        self.cam.options["quality"] = 95

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
        # Capture a frame from the camera
        frame = self.cam.capture_array()

        # Convert the frame to a texture
        buffer = frame.tobytes()
        texture = Texture.create(size=(frame.shape[1], frame.shape[0]))
        texture.blit_buffer(buffer)
        self.texture = texture

    def stop(self):
        # Stop the camera
        self.cam.stop()

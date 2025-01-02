import cv2
from kivy.graphics.texture import Texture
from kivy.uix.image import Image


class LivePreview(Image):
    def __init__(self, camera_width=1920, camera_height=1080, **kwargs):
        super().__init__(**kwargs)
        self.capture = cv2.VideoCapture(0)
        self.capture.set(cv2.CAP_PROP_FRAME_WIDTH, camera_width)
        self.capture.set(cv2.CAP_PROP_FRAME_HEIGHT, camera_height)

    def update_frame(self, dt):
        # Read a frame from the camera
        ret, frame = self.capture.read()
        if ret:
            # Convert the frame to a texture
            buffer = cv2.flip(frame, 0).tobytes()
            texture = Texture.create(size=(frame.shape[1], frame.shape[0]), colorfmt='bgr')
            texture.blit_buffer(buffer, colorfmt='bgr', bufferfmt='ubyte')
            self.texture = texture

    def stop(self):
        # Release the camera resource
        self.capture.release()

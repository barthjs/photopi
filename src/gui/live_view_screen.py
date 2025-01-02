from kivy.clock import Clock
from kivy.uix.screenmanager import Screen

from src.camera.live_preview import LivePreview


class LiveViewScreen(Screen):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.live_preview = LivePreview()
        self.add_widget(self.live_preview)

    def on_enter(self):
        # Start updating the camera frame
        Clock.schedule_interval(self.live_preview.update_frame, 1.0 / 60.0)

    def on_leave(self):
        # Stop updating and release the camera
        Clock.unschedule(self.live_preview.update_frame)
        self.live_preview.stop()

import os
from datetime import datetime

from PIL import Image
from kivy.clock import Clock
from kivy.uix.button import Button
from kivy.uix.floatlayout import FloatLayout
from kivy.uix.label import Label
from kivy.uix.screenmanager import Screen

from src.camera.live_preview import LivePreview


class LiveViewScreen(Screen):
    def __init__(self, images_config, **kwargs):
        super().__init__(**kwargs)
        self.layout = FloatLayout()
        self.images_config = images_config
        self.dir_index = None

        # Live preview
        self.live_preview = LivePreview(size_hint=(1, 1), overlay_path=images_config['preview_overlay'])
        self.layout.add_widget(self.live_preview)

        # Countdown-Label
        self.countdown = None
        self.countdown_label = Label(
            text="",
            font_size=50,
            color=(1, 0, 0, 1),
            size_hint=(None, None),
            size=(400, 100),
            pos_hint={"center": (0.5, 0.5)}
        )
        self.layout.add_widget(self.countdown_label)

        # Progress Label
        self.image_count = None
        self.max_image_count = self.images_config["max_image_count"]
        self.progress_label = Label(
            text=f"Images: 0/{self.max_image_count}",
            font_size=20,
            color=(1, 0, 0, 1),
            size_hint=(None, None),
            size=(200, 50),
            pos_hint={"x": 0.05, "y": 0.05}
        )
        self.layout.add_widget(self.progress_label)

        # Button to start countdown and capture
        self.capture_button = Button(
            text="Start Capture",
            font_size=30,
            color=(1, 1, 1, 1),
            size_hint=(None, None),
            size=(300, 100),
            pos_hint={"center_x": 0.5, "bottom": 0.1},
            background_normal='',
            background_color=(0.2, 0.6, 1, 1),
            bold=True,
            padding=[5, 5]
        )
        self.capture_button.bind(on_press=self.on_start_pressed)
        self.layout.add_widget(self.capture_button)
        self.prevent_initial_tap = True

        self.add_widget(self.layout)

    def on_start_pressed(self, instance):
        """Delay capture start slightly to prevent instant tap from previous screen."""
        if self.prevent_initial_tap:
            self.prevent_initial_tap = False
            return
        Clock.schedule_once(self.start_sequence, 0.3)

    def start_sequence(self, instance):
        """Start the sequence of 4 image captures with countdown"""
        self.set_next_dir_index()
        self.create_image_dir()
        self.image_count = 0
        self.capture_button.opacity = 0

        Clock.schedule_once(self.start_countdown, 1)

    def set_next_dir_index(self):
        """Find the next available directory index (starting from 0000)"""
        dir_index = 0
        while os.path.exists(f"server/static/images/{dir_index:04d}"):
            dir_index += 1
        self.dir_index = dir_index

    def create_image_dir(self):
        """Create a new directory for storing the images"""
        dir_name = f"server/static/images/{self.dir_index:04d}"
        os.makedirs(dir_name, exist_ok=True)
        print(f"Creating directory: {dir_name}")

    def start_countdown(self, dt):
        """Starts the countdown and captures images at intervals"""
        self.countdown = 10
        self.countdown_label.opacity = 1
        self.countdown_label.text = f"{self.countdown}"

        Clock.schedule_interval(self.update_countdown, 1)

    def update_countdown(self, dt):
        """Smooth countdown update before capturing an image."""
        self.countdown -= 1
        self.countdown_label.text = f"{self.countdown}"

        if self.countdown == 0:
            Clock.unschedule(self.update_countdown)
            self.countdown_label.opacity = 0
            self.capture_image()
            self.image_count += 1
            self.progress_label.text = f"Images: {self.image_count}/{self.max_image_count}"

            if self.image_count == self.max_image_count:
                self.countdown_label.opacity = 1
                self.end_sequence()
            else:
                Clock.schedule_once(self.start_countdown, 5)

    def capture_image(self):
        """Capture an image and save it to the corresponding folder"""
        self.live_preview.cam.switch_mode(self.live_preview.capture_config)
        frame = self.live_preview.cam.capture_array()
        if frame is not None:
            # Get timestamp for the filename
            timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
            filename = f"server/static/images/{self.dir_index:04d}/PhotoPi-{timestamp}_{self.image_count}.jpg"

            # Save the captured image
            try:
                # Convert the frame (NumPy array) to a PIL image
                pil_image = Image.fromarray(frame).rotate(180)
                pil_image.save(filename)

                print(f"Image {self.image_count} saved successfully: {filename}")
            except Exception as e:
                print(f"Error saving image {self.image_count}: {e}")
        else:
            print("Failed to capture image")
        self.live_preview.cam.switch_mode(self.live_preview.preview_config)

    def end_sequence(self):
        """End the image capture sequence and show completion message"""
        print(f"Image capture sequence completed. All images are saved in directory {self.dir_index:04d}")
        self.countdown_label.text = "Capture Complete!"
        Clock.schedule_once(self.reset_live_preview, 4)

        # Schedule transition to the email screen after 4 seconds
        Clock.schedule_once(self.return_to_email_screen, 4)

    def return_to_email_screen(self, dt):
        """Navigate to the email screen."""
        email_screen = self.manager.get_screen('email_screen')
        email_screen.set_attachment_dir(f"server/static/images/{self.dir_index:04d}")
        self.manager.current = 'email_screen'

    def reset_live_preview(self, dt):
        """Resets the screen to live preview after the image capture"""
        self.live_preview.update_frame(dt)
        self.progress_label.text = ""
        self.countdown_label.text = ""
        self.capture_button.opacity = 1

    def on_enter(self):
        """Start updating the camera frame"""
        Clock.schedule_interval(self.live_preview.update_frame, 1.0 / 30.0)

    def on_leave(self):
        """Stop updating the camera and release the resources"""
        self.prevent_initial_tap = True
        self.progress_label.text = f"Images: 0/{self.max_image_count}"
        Clock.unschedule(self.live_preview.update_frame)

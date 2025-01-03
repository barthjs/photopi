import os
from datetime import datetime

import cv2
from kivy.clock import Clock
from kivy.uix.button import Button
from kivy.uix.floatlayout import FloatLayout
from kivy.uix.label import Label
from kivy.uix.screenmanager import Screen

from src.camera.live_preview import LivePreview


class LiveViewScreen(Screen):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.layout = FloatLayout()

        # Live preview
        self.live_preview = LivePreview(size_hint=(1, 1))
        self.layout.add_widget(self.live_preview)

        # Countdown-Label
        self.countdown_label = Label(
            text="",
            font_size=50,
            color=(1, 0, 0, 1),
            size_hint=(None, None),
            size=(400, 100))
        self.countdown_label.pos_hint = {"center": (0.5, 0.5)}
        self.layout.add_widget(self.countdown_label)

        # Progress Label
        self.max_image_count = 4
        self.progress_label = Label(
            text=f"Images: 0/{self.max_image_count}",
            font_size=20,
            color=(1, 0, 0, 1),
            size_hint=(None, None),
            size=(200, 50)
        )
        self.progress_label.pos_hint = {"x": 0.05, "y": 0.05}
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
        self.capture_button.bind(on_press=self.start_sequence)
        self.layout.add_widget(self.capture_button)

        self.add_widget(self.layout)

    def start_sequence(self, instance):
        """Start the sequence of 4 image captures with countdown"""
        self.is_capturing = True
        self.dir_index = self.get_next_dir_index()
        self.image_count = 1
        self.create_image_dir()
        self.start_countdown()

        # Hide the capture button during countdown
        self.capture_button.opacity = 0

    def get_next_dir_index(self):
        """Find the next available directory index (starting from 0000)"""
        dir_index = 0
        while os.path.exists(f"server/static/images/{dir_index:04d}"):
            dir_index += 1
        return dir_index

    def create_image_dir(self):
        """Create a new directory for storing the images"""
        dir_name = f"server/static/images/{self.dir_index:04d}"
        os.makedirs(dir_name, exist_ok=True)
        print(f"Creating directory: {dir_name}")

    def update_progress_label(self):
        """Update the progress label with the current image count"""
        self.progress_label.text = f"Images: {self.image_count}/4"

    def start_countdown(self):
        """Starts the countdown and captures images at intervals"""
        self.countdown = 3
        self.countdown_label.text = f"Countdown: {self.countdown}"
        Clock.schedule_interval(self.update_countdown, 1)

    def update_countdown(self, dt):
        """Updates countdown every second and captures images"""
        self.countdown -= 1
        self.countdown_label.text = f"Countdown: {self.countdown}"

        if self.countdown <= 0:
            # Capture the current image and reset countdown
            self.capture_image()
            self.update_progress_label()
            self.image_count += 1

            # If we have captured all 4 images, stop the process
            if self.image_count > self.max_image_count:
                Clock.unschedule(self.update_countdown)
                self.end_sequence()
            else:
                # Reset countdown for next image capture
                self.countdown = 3
                self.countdown_label.text = f"Countdown: {self.countdown}"

    def capture_image(self):
        """Capture an image and save it to the corresponding folder"""
        ret, frame = self.live_preview.capture.read()
        if ret:
            # Get timestamp for the filename
            timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
            filename = f"server/static/images/{self.dir_index:04d}/PhotoPi-{timestamp}_{self.image_count}.jpg"

            # Save the captured image
            try:
                success = cv2.imwrite(filename, frame)
                if success:
                    print(f"Image {self.image_count} saved successfully: {filename}")
                else:
                    print(f"Failed to save image {self.image_count}: {filename}")
            except Exception as e:
                print(f"Error saving image {self.image_count}: {e}")
        else:
            print("Failed to capture image")

    def end_sequence(self):
        """End the image capture sequence and show completion message"""
        print(f"Image capture sequence completed. All images are saved in directory {self.dir_index:04d}")
        self.countdown_label.text = "Capture Complete!"
        # Show the button again after 2 seconds
        Clock.schedule_once(self.reset_live_preview, 2)
        Clock.schedule_once(self.show_button, 2)

        # Schedule transition to the welcome screen after 2 seconds
        Clock.schedule_once(self.return_to_welcome_screen, 2)

    def return_to_welcome_screen(self, dt):
        """Navigate back to the welcome screen."""
        self.manager.current = 'welcome_screen'

    def reset_live_preview(self, dt):
        """Resets the screen to live preview after the image capture"""
        self.live_preview.update_frame(dt)
        self.progress_label.text = ""
        self.countdown_label.text = ""

    def show_button(self, dt):
        """Show the capture button after the sequence ends"""
        self.capture_button.opacity = 1

    def on_enter(self):
        """Start updating the camera frame"""
        Clock.schedule_interval(self.live_preview.update_frame, 1.0 / 30.0)

    def on_leave(self):
        """Stop updating the camera and release the resources"""
        Clock.unschedule(self.live_preview.update_frame)

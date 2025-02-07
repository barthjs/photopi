import os
from datetime import datetime

from PIL import Image as PilImage
from kivy.clock import Clock
from kivy.uix.screenmanager import Screen


class LiveViewScreen(Screen):
    def __init__(self, images_config, **kwargs):
        super().__init__(**kwargs)
        self.images_config = images_config
        self.dir_index = None
        self.countdown = None
        self.image_count = 0
        self.inactive = True
        self.max_image_count = images_config["max_image_count"]

    def start_sequence(self, instance):
        """Start the sequence of 4 image captures with countdown"""
        self.inactive = False
        self.set_next_dir_index()
        self.create_image_dir()
        self.image_count = 0
        self.ids.capture_button.opacity = 0

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
        self.ids.countdown_label.opacity = 1
        self.ids.countdown_label.text = f"{self.countdown}"

        Clock.schedule_interval(self.update_countdown, 1)

    def update_countdown(self, dt):
        """Smooth countdown update before capturing an image."""
        self.countdown -= 1
        self.ids.countdown_label.text = f"{self.countdown}"

        if self.countdown == 0:
            Clock.unschedule(self.update_countdown)
            self.ids.countdown_label.opacity = 0
            self.capture_image()
            self.image_count += 1
            self.ids.progress_label.text = f"Images: {self.image_count}/{self.max_image_count}"

            if self.image_count == self.max_image_count:
                self.end_sequence()
            else:
                Clock.schedule_once(self.start_countdown, 5)

    def capture_image(self):
        """Capture and save an image."""
        self.ids.live_preview.cam.switch_mode(self.ids.live_preview.capture_config)
        frame = self.ids.live_preview.cam.capture_array()
        if frame is not None:
            timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
            filename = f"server/static/images/{self.dir_index:04d}/PhotoPi-{timestamp}_{self.image_count}.jpg"

            try:
                pil_image = PilImage.fromarray(frame).transpose(PilImage.FLIP_TOP_BOTTOM)

                try:
                    overlay = PilImage.open(self.images_config['final_overlay']).resize((4000, 2400)).convert("RGBA")
                    pil_image.paste(overlay, (0, 0), overlay)
                except Exception as e:
                    print(f"Warning: Could not load overlay. Error: {e}")

                # Save the image (with or without the overlay)
                pil_image.save(filename)
                print(f"Image {self.image_count} saved successfully: {filename}")
            except Exception as e:
                print(f"Error saving image {self.image_count}: {e}")
        else:
            print("Failed to capture image")

        self.ids.live_preview.cam.switch_mode(self.ids.live_preview.preview_config)

    def end_sequence(self):
        """End the image capture sequence and show completion message"""
        print(f"Image capture sequence completed. All images are saved in directory {self.dir_index:04d}")
        self.ids.countdown_label.opacity = 1
        self.ids.countdown_label.text = "Capture Complete!"

        # Schedule transition to the email screen after 4 seconds
        Clock.schedule_once(self.return_to_email_screen, 4)

    def return_to_email_screen(self, dt):
        """Navigate to the email screen."""
        email_screen = self.manager.get_screen('email_screen')
        email_screen.set_attachment_dir(f"server/static/images/{self.dir_index:04d}")
        self.manager.current = 'email_screen'

    def return_to_welcome_screen(self, dt):
        """Navigate to the email screen."""
        self.manager.current = 'welcome_screen'

    def reset_live_preview(self, dt):
        """Resets the screen to live preview after the image capture"""
        self.ids.live_preview.update_frame(dt)
        self.ids.progress_label.text = ""
        self.ids.countdown_label.text = ""
        self.ids.capture_button.opacity = 1

    def check_activity(self, dt):
        """Return to welcome screen after 60 seconds of inactivity"""
        if self.inactive:
            Clock.schedule_once(self.reset_live_preview, 1)
            Clock.schedule_once(self.return_to_welcome_screen, 1)

    def on_enter(self):
        """Start updating the camera frame"""
        self.inactive = True
        Clock.schedule_once(self.reset_live_preview, 0)
        Clock.schedule_once(self.check_activity, 60)
        Clock.schedule_interval(self.ids.live_preview.update_frame, 1.0 / 30.0)

    def on_leave(self):
        """Stop updating the camera and release the resources"""
        Clock.schedule_once(self.reset_live_preview, 0)
        Clock.unschedule(self.ids.live_preview.update_frame)

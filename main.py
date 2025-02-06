import configparser
import os

import kivy

kivy.require('2.3.1')

from kivy.config import Config

# Set window to fullscreen
Config.set('graphics', 'fullscreen', '1')
Config.set('graphics', 'borderless', '1')
Config.set('graphics', 'resizable', '0')

# Enable virtual keyboard
Config.set('kivy', 'keyboard_mode', 'systemanddock')
Config.set('kivy', 'keyboard_layout', 'src/keyboards/email.json')
Config.set('input', 'mouse', '')

from kivymd.app import MDApp
from kivy.uix.screenmanager import ScreenManager, NoTransition

from src.gui.live_view_screen import LiveViewScreen
from src.gui.welcome_screen import WelcomeScreen
from src.gui.email_screen import EmailScreen

config = configparser.ConfigParser()
config.read("config.ini")


def load_email_config():
    return {
        "SMTP_SERVER": config.get("EMAIL", "SMTP_SERVER"),
        "SMTP_PORT": config.getint("EMAIL", "SMTP_PORT"),
        "SMTP_USER": config.get("EMAIL", "SMTP_USER"),
        "SMTP_PASSWORD": config.get("EMAIL", "SMTP_PASSWORD"),
        "SENDER_EMAIL": config.get("EMAIL", "SENDER_EMAIL"),
    }


def load_images_config():
    preview_file = None
    final_file = None

    for file in os.listdir("overlays"):
        lower_file = file.lower()
        if "preview" in lower_file:
            preview_file = os.path.join("overlays", file)
        elif "final" in lower_file:
            final_file = os.path.join("overlays", file)

        if preview_file and final_file:
            break

    return {
        "max_image_count": int(config.get("IMAGES", "max_image_count")),
        "preview_overlay": preview_file if preview_file else "",
        "final_overlay": final_file if final_file else ""
    }


class PhotoPiApp(MDApp):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        # Load config
        self.email_config = load_email_config()
        self.images_config = load_images_config()

        # Create a ScreenManager to switch between screens
        self.screen_manager = ScreenManager()
        self.screen_manager.transition = NoTransition()

    def build(self):
        self.theme_cls.theme_style = "Light"
        self.theme_cls.primary_palette = "Teal"
        self.theme_cls.primary_hue = "900"

        # Add WelcomeScreen and LiveViewScreen to the ScreenManager
        self.screen_manager.add_widget(WelcomeScreen(name='welcome_screen'))
        self.screen_manager.add_widget(LiveViewScreen(name='live_view_screen', images_config=self.images_config))
        self.screen_manager.add_widget(EmailScreen(name='email_screen', email_config=self.email_config))

        return self.screen_manager


if __name__ == "__main__":
    PhotoPiApp().run()

import configparser

import kivy

kivy.require('2.3.1')

from kivy.config import Config

# Set window to fullscreen
Config.set('graphics', 'fullscreen', '1')
Config.set('graphics', 'borderless', '1')
Config.set('graphics', 'resizable', '0')

from kivy.app import App
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
    return {
        "max_image_count": int(config.get("IMAGES", "max_image_count")),
    }


class PhotoPiApp(App):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        # Load config
        self.email_config = load_email_config()
        self.images_config = load_images_config()

        # Create a ScreenManager to switch between screens
        self.screen_manager = ScreenManager()
        self.screen_manager.transition = NoTransition()

    def build(self):
        # Add WelcomeScreen and LiveViewScreen to the ScreenManager
        self.screen_manager.add_widget(WelcomeScreen(name='welcome_screen'))
        self.screen_manager.add_widget(LiveViewScreen(name='live_view_screen', images_config=self.images_config))
        self.screen_manager.add_widget(EmailScreen(name='email_screen', email_config=self.email_config))

        return self.screen_manager


if __name__ == "__main__":
    PhotoPiApp().run()

import kivy

from src.gui.live_view_screen import LiveViewScreen

kivy.require('2.3.1')

from kivy.config import Config

Config.set('kivy', 'log_level', 'warning')
Config.set('kivy', 'keyboard_mode', 'systemanddock')
Config.set('kivy', 'keyboard_layout', 'src/lang/keyboard.json')
Config.set('graphics', 'fullscreen', '1')
Config.set('input', 'mouse', '')

from kivymd.app import MDApp
from kivy.uix.screenmanager import ScreenManager, NoTransition

from src.gui.welcome_screen import WelcomeScreen
from src.gui.email_screen import EmailScreen
from src.config.config_loader import ConfigLoader


class PhotoPiApp(MDApp):
    def __init__(self, config_loader: ConfigLoader, **kwargs):
        super().__init__(**kwargs)
        self.config_loader = config_loader
        self.email_config = self.config_loader.load_email()
        self.images_config = self.config_loader.load_images()

        # Create a ScreenManager to switch between screens
        self.screen_manager = ScreenManager()
        self.screen_manager.transition = NoTransition()

    def build(self):
        self.theme_cls.theme_style = "Light"
        self.theme_cls.primary_palette = "Teal"
        self.theme_cls.primary_hue = "900"

        # Add Screens to the ScreenManager
        self.screen_manager.add_widget(WelcomeScreen(name='welcome_screen'))
        self.screen_manager.add_widget(LiveViewScreen(name='live_view_screen', images_config=self.images_config))
        self.screen_manager.add_widget(EmailScreen(name='email_screen', email_config=self.email_config))

        return self.screen_manager


if __name__ == "__main__":
    loader = ConfigLoader()
    loader.setup_language()
    PhotoPiApp(config_loader=loader).run()

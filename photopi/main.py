from threading import Thread

import kivy
from kivy.lang import Builder

from photopi.server import create_app

kivy.require('2.3.1')

from importlib import resources
from kivy.config import Config

# Kivy configuration must be set before importing other Kivy modules
Config.set('kivy', 'log_level', 'warning')
Config.set('kivy', 'keyboard_mode', 'systemanddock')
kb_path = str(resources.files('photopi').joinpath('lang/keyboard.json'))
Config.set('kivy', 'keyboard_layout', kb_path)
Config.set('graphics', 'fullscreen', '1')
Config.set('input', 'mouse', '')

from kivymd.app import MDApp
from kivy.uix.screenmanager import ScreenManager, NoTransition

from photopi.gui.welcome_screen import WelcomeScreen
from photopi.gui.email_screen import EmailScreen
from photopi.gui.live_view_screen import LiveViewScreen
from photopi.config.config_loader import ConfigLoader


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
        # Ensure custom widgets used in KV are registered before loading it
        # Import inside method to avoid cyclic imports on package init
        from photopi.camera.live_preview import LivePreview  # noqa: F401

        # Load KV from packaged resources
        kv_path = str(resources.files('photopi').joinpath('photopi.kv'))
        Builder.load_file(str(kv_path))

        self.theme_cls.theme_style = "Light"
        self.theme_cls.primary_palette = "Teal"
        self.theme_cls.primary_hue = "900"

        # Add Screens to the ScreenManager
        self.screen_manager.add_widget(WelcomeScreen(name='welcome_screen'))
        self.screen_manager.add_widget(LiveViewScreen(name='live_view_screen', images_config=self.images_config))
        self.screen_manager.add_widget(EmailScreen(name='email_screen', email_config=self.email_config))

        return self.screen_manager


def _start_flask(config_loader: ConfigLoader) -> Thread:
    """
    Start the Flask app in a background daemon thread. The thread will exit with the main process.
    """
    app = create_app(config_loader)
    config = config_loader.load_server()

    def _run():
        app.run(host=config['host'], port=config['port'], debug=False, use_reloader=False)

    t = Thread(target=_run, daemon=True)
    t.start()

    return t


def main() -> None:
    config_loader = ConfigLoader()
    config_loader.setup_language()

    if config_loader.config.getboolean("SERVER", "enabled", fallback=False):
        _start_flask(config_loader)

    PhotoPiApp(config_loader=config_loader).run()


if __name__ == '__main__':
    main()

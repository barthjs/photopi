import argparse
import os
from importlib import resources

os.environ["KIVY_NO_ARGS"] = "1"

import kivy

kivy.require('2.3.1')

from kivy.config import Config as KivyConfig
from kivy.lang import Builder

# Kivy configuration must be set before importing other Kivy modules
KivyConfig.set('kivy', 'log_level', 'warning')
KivyConfig.set('kivy', 'keyboard_mode', 'systemanddock')
KivyConfig.set('graphics', 'fullscreen', '1')
KivyConfig.set('graphics', 'show_cursor', '0')
# When using the RaspberryPi Touch Display, inputs are duplicated
# https://github.com/kivy/kivy/issues/4253
KivyConfig.set('input', 'mouse', '')

from kivymd.app import MDApp
from kivy.uix.screenmanager import ScreenManager, NoTransition

from photopi.gui.welcome_screen import WelcomeScreen
from photopi.config import AppConfig, ConfigLoader, LanguageManager
from photopi.gui.share_screen import ShareScreen
from photopi.gui.preview_screen import PreviewScreen
from photopi.gui.live_view_screen import LiveViewScreen


class PhotoPiApp(MDApp):
    def __init__(self, config: AppConfig, **kwargs):
        super().__init__(**kwargs)
        self.app_config = config
        self.screen_manager = ScreenManager(transition=NoTransition())

    def build(self):
        # Ensure custom widgets used in KV are registered before loading it
        from photopi.camera.live_preview import LivePreview  # noqa: F401

        kv_path = str(resources.files('photopi').joinpath('photopi.kv'))
        Builder.load_file(str(kv_path))

        self.theme_cls.theme_style = "Light"
        self.theme_cls.primary_palette = "Teal"
        self.theme_cls.primary_hue = "900"

        self.screen_manager.add_widget(WelcomeScreen(name='welcome_screen'))
        self.screen_manager.add_widget(LiveViewScreen(name='live_view_screen'))
        self.screen_manager.add_widget(PreviewScreen(name='preview_screen'))
        self.screen_manager.add_widget(ShareScreen(name='share_screen'))

        return self.screen_manager


def main() -> None:
    parser = argparse.ArgumentParser(description="PhotoPi")
    parser.add_argument(
        "--config",
        type=str,
        default=None,
        help="Path to config file"
    )

    args = parser.parse_args()
    config_loader = ConfigLoader(args)

    try:
        app_config = config_loader.load_config()
    except Exception as e:
        print(f"Error loading configuration: {e}")
        exit(1)

    language_manager = LanguageManager(app_config.general.language)
    language_manager.setup()

    KivyConfig.set('kivy', 'keyboard_layout', language_manager.get_keyboard_file())

    PhotoPiApp(config=app_config).run()


if __name__ == '__main__':
    main()

import builtins
from typing import Any

from kivy.uix.screenmanager import Screen
from kivymd.app import MDApp


class WelcomeScreen(Screen):
    """Entrypoint screen of the application."""

    @property
    def config(self):
        """Return the application configuration."""
        return MDApp.get_running_app().app_config

    def on_enter(self) -> None:
        """Called when the screen is entered."""
        if self.config.general.welcome_message:
            self.ids.welcome_label.text = self.config.general.welcome_message
        else:
            self.ids.welcome_label.text = builtins._("welcome_message")

    def on_start_pressed(self, instance: Any) -> None:
        """Switch to the 'live_view_screen' when the start button is pressed."""
        self.manager.current = 'live_view_screen'

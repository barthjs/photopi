from typing import Any

from kivy.uix.screenmanager import Screen


class WelcomeScreen(Screen):
    """Kivy screen for the entrypoint of the application."""

    def on_start_pressed(self, instance: Any) -> None:
        """Switch to the 'live_view_screen' when the start button is pressed."""
        self.manager.current = 'live_view_screen'

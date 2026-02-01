import builtins
from typing import Any, Dict

from kivy.uix.screenmanager import Screen


class WelcomeScreen(Screen):
    """Kivy screen for the entrypoint of the application."""

    def __init__(self, config: Dict[str, Any], **kwargs: Any) -> None:
        super().__init__(**kwargs)
        self.config = config

    def on_enter(self) -> None:
        """Called when the screen is entered."""
        custom_msg = self.config.get("general").get("welcome_message")

        if custom_msg:
            self.ids.welcome_label.text = custom_msg
        else:
            self.ids.welcome_label.text = builtins._("welcome_message")

    def on_start_pressed(self, instance: Any) -> None:
        """Switch to the 'live_view_screen' when the start button is pressed."""
        self.manager.current = 'live_view_screen'

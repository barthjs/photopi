import kivy

kivy.require('2.3.1')

from kivy.app import App
from kivy.uix.screenmanager import ScreenManager, NoTransition

from src.gui.live_view_screen import LiveViewScreen
from src.gui.welcome_screen import WelcomeScreen


class PhotoPiApp(App):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        # Create a ScreenManager to switch between screens
        self.screen_manager = ScreenManager()

    def build(self):
        # Add WelcomeScreen and LiveViewScreen to the ScreenManager
        self.screen_manager.add_widget(WelcomeScreen(name='welcome_screen'))
        self.screen_manager.add_widget(LiveViewScreen(name='live_view_screen'))
        self.screen_manager.transition = NoTransition()

        return self.screen_manager


if __name__ == "__main__":
    PhotoPiApp().run()

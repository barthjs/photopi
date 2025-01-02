from kivy.uix.screenmanager import Screen


class WelcomeScreen(Screen):

    def on_start_pressed(self, instance):
        # Switch to the live view screen
        self.manager.current = 'live_view'

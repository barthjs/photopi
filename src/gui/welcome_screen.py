from kivy.uix.boxlayout import BoxLayout
from kivy.uix.button import Button
from kivy.uix.label import Label
from kivy.uix.screenmanager import Screen


class WelcomeScreen(Screen):
    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.layout = BoxLayout(orientation='vertical', padding=20, spacing=20)

        self.label = Label(text="Welcome to PhotoPi", font_size=36)
        self.layout.add_widget(self.label)

        self.button = Button(
            text="Start",
            font_size=30,
            size_hint=(None, None),
            size=(300, 100),
            pos_hint={"center_x": 0.5, "bottom": 0.1},
            background_normal='',
            background_color=(0.2, 0.6, 1, 1),
            bold=True,
            padding=[5, 5]
        )
        self.button.bind(on_press=self.on_start_pressed)
        self.layout.add_widget(self.button)

        self.add_widget(self.layout)

    def on_start_pressed(self, instance):
        # Switch to the live view screen
        self.manager.current = 'live_view_screen'

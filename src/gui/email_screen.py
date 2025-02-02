import mimetypes
import os
import smtplib
import threading
from email.message import EmailMessage

from kivy.clock import Clock
from kivy.uix.boxlayout import BoxLayout
from kivy.uix.button import Button
from kivy.uix.label import Label
from kivy.uix.screenmanager import Screen
from kivy.uix.textinput import TextInput


class EmailScreen(Screen):
    def __init__(self, email_config, **kwargs):
        super().__init__(**kwargs)
        self.layout = BoxLayout(orientation='vertical', padding=20, spacing=20)
        self.attachment_dir = None
        self.email_config = email_config

        # Email input label
        self.label = Label(text="Enter your E-Mail", font_size=36)
        self.layout.add_widget(self.label)

        # Email input field
        self.email_input = TextInput(
            font_size=30,
            size_hint=(None, None),
            size=(600, 80),
            pos_hint={"center_x": 0.5},
            multiline=False
        )
        self.layout.add_widget(self.email_input)

        # Email send button
        self.button = Button(
            text="Send",
            font_size=30,
            size_hint=(None, None),
            size=(300, 100),
            pos_hint={"center_x": 0.5, "bottom": 0.1},
            background_normal='',
            background_color=(0.2, 0.6, 1, 1),
            bold=True,
            padding=[5, 5]
        )
        self.button.bind(on_press=self.on_send_pressed)
        self.layout.add_widget(self.button)

        self.add_widget(self.layout)

    def set_attachment_dir(self, attachment_dir):
        """Set the file paths for email attachments."""
        self.reset_screen()
        self.attachment_dir = attachment_dir

    def reset_screen(self):
        """Resets the screen UI to the initial state."""
        self.label.text = "Enter your E-Mail"
        self.email_input.text = ""
        self.email_input.opacity = 1
        self.button.opacity = 1

    def on_send_pressed(self, instance):
        """Starts the email sending process in a separate thread."""
        recipient_email = self.email_input.text.strip()
        if recipient_email:
            self.email_input.opacity = 0
            self.button.opacity = 0
            self.label.text = "Sending email..."

            threading.Thread(target=self.send_email, args=(recipient_email,), daemon=True).start()
            Clock.schedule_once(self.return_to_welcome_screen, 10)

    def send_email(self, recipient_email):
        """
        Sends an email with multiple attachments using SSL.
        Runs in a separate thread to prevent UI blocking.
        """
        try:
            msg = EmailMessage()
            msg["From"] = self.email_config["SENDER_EMAIL"]
            msg["To"] = recipient_email
            msg["Subject"] = "Your PhotoBox Images"

            # HTML Body
            html_body = "<h1>Thank you for using PhotoPi!</h1><p>Your images are attached.</p>"
            plain_body = "Thank you for using PhotoPi!\nYour images are attached."
            msg.set_content(plain_body)
            msg.add_alternative(html_body, subtype="html")

            if os.path.exists(self.attachment_dir) and os.path.isdir(self.attachment_dir):
                for filename in os.listdir(self.attachment_dir):
                    file_path = os.path.join(self.attachment_dir, filename)

                    if os.path.isfile(file_path):
                        mime_type, _ = mimetypes.guess_type(file_path)
                        mime_type = mime_type or "application/octet-stream"
                        with open(file_path, "rb") as file:
                            msg.add_attachment(file.read(),
                                               maintype=mime_type.split("/")[0],
                                               subtype=mime_type.split("/")[1],
                                               filename=os.path.basename(file_path))

            with smtplib.SMTP_SSL(self.email_config["SMTP_SERVER"], self.email_config["SMTP_PORT"]) as server:
                server.login(self.email_config["SMTP_USER"], self.email_config["SMTP_PASSWORD"])
                server.send_message(msg)

            print(f"Email successfully sent to {recipient_email}")
            Clock.schedule_once(lambda dt: self.update_label(f"Email successfully sent to {recipient_email}"), 0)

        except Exception as e:
            print(f"Error sending email to: {recipient_email} {e}")
            Clock.schedule_once(lambda dt: self.update_label(f"Error sending email to: {recipient_email}"), 0)

    def update_label(self, text):
        """Updates the label text on the main UI thread."""
        self.label.text = text

    def return_to_welcome_screen(self, dt):
        """Navigate to the welcome screen."""
        self.manager.current = 'welcome_screen'

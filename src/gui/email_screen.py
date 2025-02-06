import mimetypes
import os
import smtplib
import threading
from email.message import EmailMessage

from kivy.clock import Clock
from kivy.uix.screenmanager import Screen
from kivymd.uix.button import MDFlatButton
from kivymd.uix.dialog import MDDialog


class EmailScreen(Screen):
    def __init__(self, email_config, **kwargs):
        super().__init__(**kwargs)
        self.email_config = email_config
        self.attachment_dir = None
        self.dialog = None
        self.error_attempts = 0
        self.max_error_attempts = self.email_config["MAX_ERROR_ATTEMPTS"]

    def set_attachment_dir(self, attachment_dir):
        """Set the file paths for email attachments."""
        self.attachment_dir = attachment_dir

    def on_send_pressed(self, instance):
        """Starts the email sending process in a separate thread."""
        recipient_email = self.ids.email_input.text.strip()
        if recipient_email:
            self.ids.email_label.text = "Sending email..."
            self.ids.email_input.text = ""
            self.ids.email_input.opacity = 0
            self.ids.email_send.opacity = 0
            self.ids.email_abort.opacity = 0

            threading.Thread(target=self.send_email, args=(recipient_email,), daemon=True).start()
        else:
            self.ids.email_label.text = "Please enter a valid email address."

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

            self.error_attempts = 0
            print(f"Email successfully sent to {recipient_email}")
            Clock.schedule_once(lambda dt: self.update_label(f"Email successfully sent to {recipient_email}"), 0)
            Clock.schedule_once(self.return_to_welcome_screen, 10)

        except Exception as e:
            print(f"Error sending email to: {recipient_email} {e}")
            self.error_attempts += 1
            self.log_email_attempt(
                f"Error sending email to: {recipient_email} - Attempt {self.error_attempts}/{self.max_error_attempts}")
            if self.error_attempts < self.max_error_attempts:
                Clock.schedule_once(lambda dt: self.update_label(
                    f"Error sending email to: {recipient_email} (Attempts: {self.error_attempts}/{self.max_error_attempts})"),
                                    0)
                Clock.schedule_once(self.reset_screen_after_error, 0)
            else:
                # Log error limit reached
                self.log_email_attempt("Error limit reached for sending emails.")
                Clock.schedule_once(lambda dt: self.update_label(
                    f"Error limit reached. Please contact: {self.email_config['ADMIN_EMAIL']}"), 0)
                Clock.schedule_once(self.return_to_welcome_screen, 30)

    def log_email_attempt(self, message):
        """Log email attempt or error in the attachment directory."""
        log_file_path = os.path.join(self.attachment_dir, "email_attempts_log.txt")
        with open(log_file_path, "a") as log_file:
            log_file.write(f"{message}\n")

    def update_label(self, text):
        """Updates the label text on the main UI thread."""
        self.ids.email_label.text = text

    def on_abort_pressed(self):
        """Handles the abort button press and shows a confirmation dialog."""
        if self.dialog:
            self.dialog.dismiss()
        self.dialog = MDDialog(
            title="Confirm abort",
            text="Are you sure you want to cancel sending the email?",
            buttons=[
                MDFlatButton(text="Cancel", on_press=self.dismiss_dialog),
                MDFlatButton(text="Confirm", on_press=self.abort_email_send)
            ]
        )
        self.dialog.open()

    def dismiss_dialog(self, instance):
        """Dismiss the dialog."""
        self.dialog.dismiss()

    def abort_email_send(self, instance):
        """Handle the abort action, and log user action."""
        self.dialog.dismiss()
        self.ids.email_input.text = ""
        self.ids.email_label.text = "Email sending aborted."

        # Log to file
        self.log_email_attempt("User aborted the email sending process.")

        # Reset the attempts and show the welcome screen
        self.error_attempts = 0
        self.manager.current = 'welcome_screen'

    def reset_screen_after_error(self, dt):
        self.ids.email_input.opacity = 1
        self.ids.email_send.opacity = 1
        self.ids.email_abort.opacity = 1

    def return_to_welcome_screen(self, dt):
        """Navigate to the welcome screen."""
        self.manager.current = 'welcome_screen'

    def on_enter(self):
        """Resets the screen UI to the initial state."""
        self.ids.email_label.text = "Enter your E-Mail"
        self.ids.email_input.text = ""
        self.ids.email_input.opacity = 1
        self.ids.email_send.opacity = 1
        self.ids.email_abort.opacity = 1

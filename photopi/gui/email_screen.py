import builtins
import os
import smtplib
import threading
from email.message import EmailMessage
from typing import Any, Dict, Optional

from kivy.clock import Clock
from kivy.uix.screenmanager import Screen
from kivymd.uix.button import MDFlatButton
from kivymd.uix.dialog import MDDialog


# noinspection PyProtectedMember,PyUnresolvedReferences,PyUnusedLocal
class EmailScreen(Screen):
    """Kivy screen that handles sending images via email."""

    def __init__(self, config: Dict[str, Any], **kwargs: Any) -> None:
        super().__init__(**kwargs)
        self.config = config
        self.email_config: Dict[str, str | int | bool] = config["email"]
        self.general_config = config["general"]
        self.attachment_dir: Optional[str] = None
        self.dialog: Optional[MDDialog] = None
        self.attempts: int = 0
        self.max_attempts: int = 5
        self.is_sending: bool = False

    def set_attachment_dir(self, attachment_dir: str) -> None:
        """Set the directory containing attachments to send."""
        self.attachment_dir = attachment_dir

    def on_send_pressed(self, instance: Any) -> None:
        """Starts the email sending process in a separate thread."""
        if self.is_sending:
            return

        recipient_email = self.ids.email_input.text.strip()
        if "@" in recipient_email and 254 >= len(recipient_email) >= 6:
            self.is_sending = True
            self.hide_input_fields()
            self.update_label(builtins._("Sending email..."))

            Clock.schedule_once(
                lambda dt: threading.Thread(target=self._send_email, args=(recipient_email,), daemon=True).start())
        else:
            self.update_label(builtins._("Please enter a valid email address."))

    def _send_email(self, recipient_email):
        """
        Sends an email with the captured images as attachments.
        Runs in a separate thread to prevent UI blocking.
        """
        try:
            msg = self._create_email_message(recipient_email)
            self._attach_images(msg)

            with smtplib.SMTP_SSL(
                self.email_config["smtp_server"],
                self.email_config["smtp_port"]
            ) as server:
                server.login(
                    self.email_config["smtp_user"],
                    self.email_config["smtp_password"]
                )
                server.send_message(msg)

            Clock.schedule_once(
                lambda dt: self.update_label(
                    builtins._("Email successfully sent to {}").format(recipient_email)
                )
            )
            Clock.schedule_once(self.return_to_welcome_screen, 10)

        except Exception as e:
            self.attempts += 1
            self._log_email_attempt(
                f"Error sending email to: {recipient_email} - Attempts: {self.attempts}/{self.max_attempts} - Error: {e}"
            )

            if self.attempts < self.max_attempts:
                # Display an error and show input again
                Clock.schedule_once(
                    lambda dt: self.update_label(
                        builtins._("Error sending email to: {} (Attempts: {}/{})").format(
                            recipient_email, self.attempts, self.max_attempts
                        )
                    )
                )
                Clock.schedule_once(self.show_input_fields)
            else:
                # Display an error and return to the welcome screen
                Clock.schedule_once(
                    lambda dt: self.update_label(
                        builtins._("Error limit reached. Please contact: {}").format(
                            self.email_config["admin_email"]
                        )
                    )
                )
                Clock.schedule_once(self.return_to_welcome_screen, 60)
        finally:
            self.is_sending = False

    def _create_email_message(self, recipient_email: str) -> EmailMessage:
        """Create and return the email message object."""
        msg = EmailMessage()
        msg["From"] = self.email_config["sender_email"]
        msg["To"] = recipient_email
        msg["Subject"] = builtins._("Your {} Images").format(self.general_config["name"])

        thank_you = builtins._("Thank you for using {}!").format(self.general_config["name"])
        html_body = f"<h1>{thank_you}</h1><p>{builtins._('Your images are attached.')}</p>"
        plain_body = f"{thank_you}\n{builtins._('Your images are attached.')}"
        msg.set_content(plain_body)
        msg.add_alternative(html_body, subtype="html")
        return msg

    def _attach_images(self, msg: EmailMessage) -> None:
        """Attach all .jpg images from the attachment directory."""
        if not self.attachment_dir:
            return

        for filename in os.listdir(self.attachment_dir):
            file_path = os.path.join(self.attachment_dir, filename)
            if os.path.isfile(file_path) and filename.lower().endswith(".jpg"):
                with open(file_path, "rb") as file:
                    msg.add_attachment(
                        file.read(),
                        maintype="image",
                        subtype="jpeg",
                        filename=os.path.basename(file_path)
                    )

    def _log_email_attempt(self, message: str) -> None:
        """Log email sending attempts in the image directory."""
        if not self.attachment_dir:
            return
        log_file_path = os.path.join(self.attachment_dir, "log.txt")
        with open(log_file_path, "a", encoding="utf-8") as log_file:
            log_file.write(f"{message}\n")

    def update_label(self, text: str) -> None:
        """Update the label text on the UI thread."""
        self.ids.email_label.text = text

    def on_abort_pressed(self) -> None:
        """Show a confirmation dialog to abort sending the email."""
        if self.dialog:
            self.dialog.dismiss()

        self.dialog = MDDialog(
            title=builtins._("Confirm abort"),
            text=builtins._("Are you sure you want to cancel sending the email?"),
            buttons=[
                MDFlatButton(text=builtins._("Cancel"), on_press=self.dismiss_dialog),
                MDFlatButton(text=builtins._("Confirm"), on_press=self.abort_email_send)
            ]
        )
        self.dialog.open()

    def dismiss_dialog(self, instance: Any) -> None:
        """Dismiss the abort confirmation dialog."""
        if self.dialog:
            self.dialog.dismiss()

    def abort_email_send(self, instance: Any) -> None:
        """Abort the email sending process and return to the welcome screen."""
        if self.dialog:
            self.dialog.dismiss()
        self.return_to_welcome_screen()

    def show_input_fields(self, dt: Optional[float] = None) -> None:
        """Make the email input fields visible."""
        self.ids.email_input.opacity = 1
        self.ids.email_input.disabled = False
        self.ids.email_send.opacity = 1
        self.ids.email_send.disabled = False
        self.ids.email_abort.opacity = 1
        self.ids.email_abort.disabled = False

    def hide_input_fields(self, dt: Optional[float] = None) -> None:
        """Make the email input fields invisible."""
        self.ids.email_input.opacity = 0
        self.ids.email_input.disabled = True
        self.ids.email_send.opacity = 0
        self.ids.email_send.disabled = True
        self.ids.email_abort.opacity = 0
        self.ids.email_abort.disabled = True

    def return_to_welcome_screen(self, dt: Optional[float] = None) -> None:
        """Navigate back to the welcome screen."""
        self.manager.current = "welcome_screen"

    def on_enter(self) -> None:
        """Reset the screen state when it is entered."""
        if self.email_config["enabled"]:
            self.show_input_fields()
            self.update_label(builtins._("Enter your E-Mail"))
            self.ids.email_input.text = ""
            self.attempts = 0
            return

        self.update_label(builtins._("Email sending is disabled."))
        self.hide_input_fields()
        Clock.schedule_once(self.return_to_welcome_screen, 10)

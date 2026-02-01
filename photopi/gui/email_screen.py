import builtins
import os
import re
import smtplib
import threading
import traceback
from datetime import datetime
from email.message import EmailMessage
from typing import Any, Dict, Optional

from jinja2 import Environment, PackageLoader, select_autoescape
from kivy.clock import Clock
from kivy.properties import BooleanProperty
from kivy.uix.screenmanager import Screen
from kivymd.uix.button import MDFlatButton
from kivymd.uix.dialog import MDDialog


class EmailScreen(Screen):
    """Kivy screen that handles sending images via email."""
    ui_active = BooleanProperty(True)
    is_sending = BooleanProperty(False)

    def __init__(self, config: Dict[str, Any], **kwargs: Any) -> None:
        super().__init__(**kwargs)
        self.config: Dict[str, Any] = config
        self.email_config: Dict[str, str | int | bool] = config["email"]
        self.general_config: Dict[str, Any] = config["general"]
        self.attachment_dir: Optional[str] = None
        self.attempts: int = 0
        self.max_attempts: int = 5
        self.dialog: Optional[MDDialog] = None

        self.email_pattern: re.Pattern = re.compile(r"^[a-zA-Z0-9_.+-]+@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+$")

        self.jinja_env: Environment = Environment(
            loader=PackageLoader("photopi", "templates"),
            autoescape=select_autoescape(["html", "xml"])
        )

    def set_attachment_dir(self, attachment_dir: str) -> None:
        """Set the directory containing attachments to send."""
        self.attachment_dir = attachment_dir

    def on_enter(self) -> None:
        """Prepare screen on entry."""
        if self.email_config.get("enabled"):
            self.ids.email_input.text = ""
            self._update_label(builtins._("email_prompt"))
            self._show_input_fields()
            self.attempts = 0
            return

        self._update_label(builtins._("email_disabled"), is_error=True)
        self._hide_input_fields()

        Clock.schedule_once(self._return_to_welcome_screen, 10)

    def on_send_pressed(self, instance: Any) -> None:
        """Starts the email sending process in a separate thread if validation passes."""
        if self.is_sending:
            return

        recipient_email: str = self.ids.email_input.text.strip()

        if self.email_pattern.match(recipient_email):
            self.is_sending = True
            self._update_label(builtins._("email_sending"))
            self._hide_input_fields()

            Clock.schedule_once(
                lambda dt: threading.Thread(target=self._send_email, args=(recipient_email,), daemon=True).start()
            )
        else:
            self._update_label(builtins._("email_invalid"), is_error=True)

    def on_abort_pressed(self) -> None:
        """Show a confirmation dialog to abort sending the email."""
        if self.dialog:
            self.dialog.dismiss()

        self.dialog = MDDialog(
            title=builtins._("abort_dialog_title"),
            text=builtins._("abort_dialog_text"),
            buttons=[
                MDFlatButton(text=builtins._("abort_dialog_back"), on_press=self._dismiss_dialog),
                MDFlatButton(text=builtins._("abort_dialog_confirm"), on_press=self._abort_email_send)
            ]
        )
        self.dialog.open()

    def _update_label(self, text: str, is_error: bool = False) -> None:
        """Update the label text and color on the UI thread."""
        self.ids.email_label.text = text

        if is_error:
            self.ids.email_label.text_color = (0.8, 0, 0, 1)
        else:
            from kivy.app import App
            self.ids.email_label.text_color = App.get_running_app().theme_cls.primary_color

    def _return_to_welcome_screen(self, dt: Optional[float] = None) -> None:
        self.manager.current = "welcome_screen"

    def _show_input_fields(self, dt: Optional[float] = None) -> None:
        """Reset UI to interactive state."""
        self.ui_active = True

    def _hide_input_fields(self, dt: Optional[float] = None) -> None:
        """Hide input fields and buttons."""
        self.ui_active = False
        self.ids.email_input.focus = False

    def _send_email(self, recipient_email: str) -> None:
        """Sends an email with all images as attachments and handles UI updates via Clock."""
        try:
            msg = self._create_email_message(recipient_email)
            self._attach_images(msg)

            with smtplib.SMTP_SSL(
                self.email_config["smtp_server"],
                self.email_config["smtp_port"],
                timeout=20
            ) as server:
                server.login(
                    self.email_config["smtp_user"],
                    self.email_config["smtp_password"]
                )
                server.send_message(msg)

            self._log_email_attempt(f"SUCCESS: Email sent to {recipient_email}")

            Clock.schedule_once(
                lambda dt: self._update_label(
                    builtins._("email_success").format(recipient_email)
                )
            )
            Clock.schedule_once(self._return_to_welcome_screen, 10)

        except Exception as e:
            self.attempts += 1

            timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            error_stack = traceback.format_exc()

            log_message = (
                f"{'#' * 50}\n"
                f"TIMESTAMP: {timestamp}\n"
                f"STATUS: Failure during attempt {self.attempts}/{self.max_attempts}\n"
                f"RECIPIENT: {recipient_email}\n"
                f"ERROR TYPE: {type(e).__name__}\n"
                f"MESSAGE: {str(e)}\n"
                f"TRACEBACK:\n{error_stack}"
                f"{'#' * 50}\n"
            )

            self._log_email_attempt(log_message)

            if self.attempts < self.max_attempts:
                # Show an error message and allow retry
                Clock.schedule_once(
                    lambda dt: self._update_label(
                        builtins._("email_error").format(
                            recipient_email, self.attempts, self.max_attempts
                        ), is_error=True
                    )
                )
                Clock.schedule_once(lambda dt: self._show_input_fields())
            else:
                # Final failure after max attempts. Show an error message and return to the welcome screen
                Clock.schedule_once(
                    lambda dt: self._update_label(
                        builtins._("email_limit").format(
                            self.email_config["admin_email"]
                        ), is_error=True
                    )
                )
                Clock.schedule_once(self._return_to_welcome_screen, 60)
        finally:
            Clock.schedule_once(lambda dt: setattr(self, 'is_sending', False))

    def _create_email_message(self, recipient_email: str) -> EmailMessage:
        """Create the EmailMessage using rendered Jinja2 templates."""
        msg = EmailMessage()
        msg["From"] = self.email_config.get("sender_email")
        msg["To"] = recipient_email

        # Prioritize config values, fallback to translation keys
        msg["Subject"] = self.email_config.get("subject") or builtins._("email_subject")

        context: Dict[str, str] = {
            "language": self.general_config.get("language"),
            "headline": self.email_config.get("headline") or builtins._("email_headline"),
            "body_text": self.email_config.get("body") or builtins._("email_body"),
            "footer_text": self.email_config.get("footer")
        }

        # Render both text and HTML versions
        text_body = self.jinja_env.get_template("email.txt.j2").render(**context)
        html_body = self.jinja_env.get_template("email.html.j2").render(**context)

        msg.set_content(text_body)
        msg.add_alternative(html_body, subtype="html")

        return msg

    def _attach_images(self, msg: EmailMessage) -> None:
        """Attach all .jpg files from the capture directory."""
        if not self.attachment_dir or not os.path.exists(self.attachment_dir):
            return

        for filename in os.listdir(self.attachment_dir):
            file_path = os.path.join(self.attachment_dir, filename)
            if os.path.isfile(file_path) and filename.lower().endswith(".jpg"):
                with open(file_path, "rb") as file:
                    msg.add_attachment(
                        file.read(),
                        maintype="image",
                        subtype="jpeg",
                        filename=filename
                    )

    def _log_email_attempt(self, message: str) -> None:
        """Log attempt details to a local file in the capture directory."""
        if self.attachment_dir:
            log_path = os.path.join(self.attachment_dir, "log.txt")
            with open(log_path, "a", encoding="utf-8") as f:
                f.write(f"{message}\n")

    def _dismiss_dialog(self, instance: Any) -> None:
        if self.dialog:
            self.dialog.dismiss()

    def _abort_email_send(self, instance: Any) -> None:
        if self.dialog:
            self.dialog.dismiss()
        self._return_to_welcome_screen()

import builtins
import os
import re
import threading
import traceback
from datetime import datetime
from typing import Any, Optional

import qrcode
from kivy.clock import Clock
from kivy.properties import BooleanProperty, StringProperty
from kivy.uix.screenmanager import Screen
from kivymd.app import MDApp
from kivymd.uix.button import MDFlatButton
from kivymd.uix.dialog import MDDialog

from photopi.sharing.email_service import EmailService
from photopi.sharing.factory import SharingFactory


class ShareScreen(Screen):
    """
    Final screen to share the photos via email and/or a cloud storage provider.
    """

    ui_active = BooleanProperty(True)
    is_sending = BooleanProperty(False)
    show_email = BooleanProperty(False)
    show_cloud = BooleanProperty(False)
    cloud_link = StringProperty("")
    qr_code_path = StringProperty("")

    def __init__(self, **kwargs: Any) -> None:
        super().__init__(**kwargs)
        self.attachment_dir: Optional[str] = None
        self.attempts: int = 0
        self.max_attempts: int = 3
        self.dialog: Optional[MDDialog] = None
        self._email_service: Optional[EmailService] = None

    @property
    def config(self):
        """Return the application configuration."""
        return MDApp.get_running_app().app_config

    def set_attachment_dir(self, attachment_dir: str) -> None:
        """Set the directory containing attachments to send."""
        self.attachment_dir = attachment_dir

    def on_enter(self) -> None:
        """Prepare screen on entry."""
        self.ui_active = True
        self.is_sending = False
        self.attempts = 0

        self.cloud_link = ""
        self.qr_code_path = ""

        self.show_email = self.config.email.enabled
        self.show_cloud = self.config.general.cloud_provider is not None

        if self.show_email:
            self.ids.email_input.text = ""
            self._update_email_label(builtins._("share_email_prompt"))
            self._email_service = EmailService(self.config.email, self.config.general.language)
        else:
            self._update_email_label("")

        if self.show_cloud:
            self._upload_to_cloud()
        elif not self.show_email:
            # Show a message if neither email nor cloud provider is configured
            # and return to the welcome screen
            self._update_email_label(builtins._("share_disabled"))
            Clock.schedule_once(self._return_to_welcome_screen, 30)

    def _upload_to_cloud(self) -> None:
        """Starts cloud upload in a separate thread."""
        self.is_sending = True
        self.ui_active = False
        threading.Thread(target=self._perform_cloud_upload, daemon=True).start()

    def _perform_cloud_upload(self) -> None:
        """Cloud upload running in a background thread."""
        try:
            provider = SharingFactory.get_cloud_provider(self.config)
            if provider and self.attachment_dir:
                link = provider.upload_files(self.attachment_dir, self.config.images.file_prefix)
                if link:
                    Clock.schedule_once(lambda dt: self._on_cloud_success(link))
                else:
                    Clock.schedule_once(lambda dt: self._on_cloud_failure())
            else:
                Clock.schedule_once(lambda dt: self._on_cloud_failure())
        except Exception as e:
            print(f"Error uploading files to cloud provider: {e}")
            Clock.schedule_once(lambda dt: self._on_cloud_failure())

    def _on_cloud_success(self, link: str) -> None:
        if self.attachment_dir:
            try:
                qr = qrcode.QRCode(
                    version=1,
                    error_correction=qrcode.constants.ERROR_CORRECT_L,
                    box_size=10,
                    border=0,
                )
                qr.add_data(link)
                qr.make(fit=True)

                img = qr.make_image(fill_color="black", back_color="transparent")
                qr_path = os.path.join(self.attachment_dir, "qr_code.png")
                img.save(qr_path)
                self.qr_code_path = qr_path
            except Exception as e:
                print(f"Error generating QR code: {e}")

        self.cloud_link = link
        self.is_sending = False
        self.ui_active = True

    def _on_cloud_failure(self) -> None:
        self.show_cloud = False
        self.is_sending = False
        self.ui_active = True

    def on_finish_pressed(self) -> None:
        self.dialog = MDDialog(
            title=builtins._("share_finish_dialog_title"),
            text=builtins._("share_finish_dialog_text"),
            buttons=[
                MDFlatButton(
                    text=builtins._("share_finish_dialog_back"),
                    on_release=self._dismiss_dialog
                ),
                MDFlatButton(
                    text=builtins._("share_finish_dialog_confirm"),
                    on_release=self._finish_sharing
                ),
            ],
        )
        self.dialog.open()

    def on_send_pressed(self, instance: Any) -> None:
        """Starts the email sending process if validation passes."""
        recipient_email = self.ids.email_input.text.strip()
        if not re.match(r"[^@]+@[^@]+\.[^@]+", recipient_email):
            self._update_email_label(builtins._("share_email_invalid"), is_error=True)
            return

        self.is_sending = True
        self.ui_active = False
        self.ids.email_input.focus = False
        self._update_email_label(builtins._("share_email_sending"))

        Clock.schedule_once(
            lambda dt: threading.Thread(target=self._send_email, args=(recipient_email,), daemon=True).start()
        )

    def _update_email_label(self, text: str, is_error: bool = False) -> None:
        self.ids.email_label.text = text

        if is_error:
            self.ids.email_label.text_color = (0.8, 0, 0, 1)
        else:
            self.ids.email_label.text_color = MDApp.get_running_app().theme_cls.primary_color

    def _return_to_welcome_screen(self, dt: Optional[float] = None) -> None:
        """Return to the welcome screen after sending the email or to many failed attempts."""
        self.manager.current = "welcome_screen"

    def _send_email(self, recipient_email: str) -> None:
        """Sends the picture to an email with the email service."""
        if not self._email_service or not self.attachment_dir:
            return

        try:
            self._email_service.send_email(recipient_email, self.attachment_dir)
            self._email_service.log_attempt(self.attachment_dir, f"SUCCESS: Email sent to {recipient_email}")

            Clock.schedule_once(
                lambda dt: self._update_email_label(
                    builtins._("share_email_success").format(recipient_email)
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
            self._email_service.log_attempt(self.attachment_dir, log_message)

            if self.attempts < self.max_attempts:
                # Show an error message and allow retry
                Clock.schedule_once(
                    lambda dt: self._update_email_label(
                        builtins._("share_email_error").format(
                            recipient_email, self.attempts, self.max_attempts
                        ), is_error=True
                    )
                )
                Clock.schedule_once(lambda dt: setattr(self, 'ui_active', True))
            else:
                # Final failure after max attempts. Show an error message and return to the welcome screen
                Clock.schedule_once(
                    lambda dt: self._update_email_label(
                        builtins._("share_email_limit").format(
                            self.config.email.admin_email
                        ), is_error=True
                    )
                )
                Clock.schedule_once(self._return_to_welcome_screen, 60)
        finally:
            Clock.schedule_once(lambda dt: setattr(self, 'is_sending', False))

    def _dismiss_dialog(self, instance: Any) -> None:
        if self.dialog:
            self.dialog.dismiss()

    def _finish_sharing(self, instance: Any) -> None:
        if self.dialog:
            self.dialog.dismiss()
        self._return_to_welcome_screen()

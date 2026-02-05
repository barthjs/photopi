import builtins
import os
import smtplib
from email.message import EmailMessage
from typing import Dict

from jinja2 import Environment, PackageLoader, select_autoescape

from photopi.config.models import EmailConfig


class EmailService:
    """Handles the logic for sending images via email using Jinja2 templates."""

    def __init__(self, config: EmailConfig, language: str):
        self.config = config
        self.language = language
        self.jinja_env = Environment(
            loader=PackageLoader("photopi", "templates"),
            autoescape=select_autoescape(["html", "xml"]),
        )

    def send_email(self, recipient_email: str, attachment_dir: str) -> None:
        """Sends an email with all images as attachments."""
        msg = self._create_email_message(recipient_email)
        self._attach_images(msg, attachment_dir)

        with smtplib.SMTP_SSL(
            self.config.smtp_server,
            self.config.smtp_port,
            timeout=20
        ) as server:
            server.login(
                self.config.smtp_user,
                self.config.smtp_password
            )
            server.send_message(msg)

    def log_attempt(self, attachment_dir: str, message: str) -> None:
        """Log attempt details to a local file in the capture directory."""
        if not attachment_dir or not os.path.exists(attachment_dir):
            return

        log_file = os.path.join(attachment_dir, "email_log.txt")
        with open(log_file, "a") as f:
            f.write(message + "\n")

    def _create_email_message(self, recipient_email: str) -> EmailMessage:
        """Create the EmailMessage using rendered Jinja2 templates."""
        msg = EmailMessage()
        msg["From"] = self.config.sender_email
        msg["To"] = recipient_email

        # Prioritize config values, fallback to translation keys
        msg["Subject"] = self.config.subject or builtins._("email_subject")

        context: Dict[str, str] = {
            "language": self.language,
            "headline": self.config.headline or builtins._("email_headline"),
            "body_text": self.config.body or builtins._("email_body"),
            "footer_text": self.config.footer
        }

        # Render both text and HTML versions
        text_body = self.jinja_env.get_template("email.txt.j2").render(**context)
        html_body = self.jinja_env.get_template("email.html.j2").render(**context)

        msg.set_content(text_body)
        msg.add_alternative(html_body, subtype="html")

        return msg

    def _attach_images(self, msg: EmailMessage, attachment_dir: str) -> None:
        """Attach all .jpg files from the capture directory."""
        if not attachment_dir or not os.path.exists(attachment_dir):
            return

        for filename in os.listdir(attachment_dir):
            file_path = os.path.join(attachment_dir, filename)
            if os.path.isfile(file_path) and filename.lower().endswith(".jpg"):
                with open(file_path, "rb") as file:
                    msg.add_attachment(
                        file.read(),
                        maintype="image",
                        subtype="jpeg",
                        filename=filename
                    )

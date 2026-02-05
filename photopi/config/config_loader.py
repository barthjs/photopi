import argparse
import configparser
from pathlib import Path
from typing import Optional

from platformdirs import user_config_dir

from photopi.config.models import AppConfig, EmailConfig, GeneralConfig, ImageConfig, NextcloudConfig


class ConfigLoader:
    """
    Loads the application configuration from an INI file.
    """

    def __init__(self, args: Optional[argparse.Namespace] = None):
        self.user_config_dir = Path(user_config_dir("photopi"))

        custom_path = getattr(args, "config", None) if args else None
        self.config_path = self._resolve_config_path(custom_path)

        self.parser = configparser.ConfigParser()

    def load_config(self) -> AppConfig:
        """
        Reads the INI file and validates it via Pydantic.
        """
        self.parser.read(self.config_path, encoding="utf-8")

        def get_section(name: str) -> dict:
            """Helper to safely extract a section as a dict"""
            return dict(self.parser[name]) if name in self.parser else {}

        return AppConfig(
            general=GeneralConfig(**get_section("GENERAL")),
            images=ImageConfig(**get_section("IMAGES")),
            email=EmailConfig(**get_section("EMAIL")),
            nextcloud=NextcloudConfig(**get_section("NEXTCLOUD"))
        )

    def _resolve_config_path(self, custom_path: Optional[str]) -> Path:
        """
        Resolves the config file path with priority:
        1. Explicit path passed to the program via the --config parameter.
        2. 'config.ini' in the current working directory.
        3. 'config.ini' in the users config directory.

        If none exists, creates the default config the users config directory.
        """
        if custom_path:
            return Path(custom_path)

        cwd_config = Path.cwd() / "config.ini"
        if cwd_config.is_file():
            return cwd_config

        user_config = self.user_config_dir / "config.ini"
        if user_config.is_file():
            return user_config

        self._create_default_config(user_config)
        return user_config

    def _create_default_config(self, path: Optional[Path] = None):
        """Writes the default configuration values to the specified path."""
        if not path:
            path = self.user_config_dir / "config.ini"

        path.parent.mkdir(parents=True, exist_ok=True)

        default_content = (
            "[GENERAL]\n"
            "name = PhotoPi\n"
            "language = en\n\n"
            "cloud_provider = none\n\n"
            "[IMAGES]\n"
            "base_image_dir = ~/.local/share/photopi/images\n"
            "max_image_count = 4\n"
            "file_prefix = PhotoPi\n\n"
            "[EMAIL]\n"
            "enabled = true\n"
            "smtp_server = \n"
            "smtp_port = 587\n"
            "smtp_user = \n"
            "smtp_password = \n"
            "sender_email = \n"
            "admin_email = \n\n"
            "[NEXTCLOUD]\n"
            "url = \n"
            "username = \n"
            "password = \n"
            "folder = \n"
        )

        try:
            path.write_text(default_content, encoding="utf-8")
        except OSError as e:
            print(f"Warning: Could not create default config at {path}: {e}")

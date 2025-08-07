import builtins
import configparser
import json
import locale
import os
from importlib import resources
from typing import Dict, Optional, Tuple


class ConfigLoader:
    """Encapsulates loading and accessing application configuration.

    - Reads config.ini from the program's current working directory if present; otherwise from ~/.config/photopi.
    - If no config.ini exists, creates ~/.config/photopi and writes a sample config.ini with defaults.
    - Provides helpers to set up language (gettext), load email and image-related config.
    """

    def __init__(self, config_path: Optional[str] = None) -> None:
        # Program root = directory from which the program is executed
        self._program_root = os.getcwd()
        self._user_config_dir = os.path.join(os.path.expanduser("~"), ".config", "photopi")

        # Decide which config.ini to use, possibly creating a sample in the user config dir
        resolved_config_path = self._resolve_or_create_config_path(config_path)

        self.config = configparser.ConfigParser()
        self.config.read(resolved_config_path)

    def _resolve_or_create_config_path(self, config_path: Optional[str]) -> str:
        """
        Resolve the config.ini path with priority: provided > program root > user.
        If none exists, create a sample config.ini in the user config dir and return its path.
        """
        if config_path:
            return config_path

        program_config = os.path.join(self._program_root, "config.ini")
        if os.path.exists(program_config):
            return program_config

        user_config = os.path.join(self._user_config_dir, "config.ini")
        if os.path.exists(user_config):
            return user_config

        # Ensure the user config dir exists and create sample config.ini
        os.makedirs(self._user_config_dir, exist_ok=True)
        with open(user_config, "w", encoding="utf-8") as f:
            f.write(self._fallback_default_config())

        return user_config

    @staticmethod
    def _fallback_default_config() -> str:
        """Hardcoded minimal default config in case config.ini is missing."""
        return (
            "[GENERAL]\n"
            "language = en\n\n"
            "[EMAIL]\n"
            "enabled = true\n"
            "smtp_server = \n"
            "smtp_port = \n"
            "smtp_user = \n"
            "smtp_password = \n"
            "sender_email = \n"
            "admin_email = \n\n"
            "[IMAGES]\n"
            "base_image_dir = ~/.local/share/photopi/images\n"
            "max_image_count = 4\n\n"
            "[SERVER]\n"
            "enabled = false\n"
            "host = *\n"
            "port = 8080\n"
        )

    def setup_language(self) -> None:
        """Sets up JSON-based translation and installs builtins._ for KV/Python."""
        lang = self.config.get("GENERAL", "language", fallback="").lower()

        if lang not in ("en", "de"):
            # Fallback to system default locale, supported: en, de
            system_lang, _ = locale.getdefaultlocale()
            if system_lang:
                lang_code = system_lang.split("_")[0].lower()
                if lang_code in ("en", "de"):
                    lang = lang_code
                else:
                    lang = "en"
            else:
                lang = "en"

        translations = self._load_translations(lang)

        def _(text: str) -> str:
            return translations.get(text, text)

        builtins._ = _

    @staticmethod
    def _load_translations(lang: str) -> dict[str, str]:
        """Load translations from JSON with fallback to English."""
        lang_dir = str(resources.files("photopi").joinpath("lang"))

        def read_json(path: str) -> dict[str, str]:
            try:
                with open(path, encoding="utf-8") as f:
                    data = json.load(f)
                    return {str(k): str(v) for k, v in data.items() if isinstance(data, dict)}
            except FileNotFoundError:
                return {}
            except Exception:
                return {}

        fallback = read_json(os.path.join(lang_dir, "en.json"))
        primary = read_json(os.path.join(lang_dir, f"{lang}.json"))

        # primary overrides fallback
        return {**fallback, **primary}

    def load_email(self) -> Dict[str, str | int | bool]:
        """Returns email-related configuration."""
        try:
            enabled = self.config.getboolean("EMAIL", "enabled", fallback=False)
        except ValueError:
            enabled = False

        if not enabled:
            return {"enabled": False}

        email_config = {}
        keys = [
            "smtp_server",
            "smtp_port",
            "smtp_user",
            "smtp_password",
            "sender_email",
            "admin_email",
        ]

        for key in keys:
            value = self.config.get("EMAIL", key, fallback="").strip()
            if not value:
                return {"enabled": False}
            email_config[key] = value

        try:
            email_config["smtp_port"] = int(email_config["smtp_port"])
        except ValueError:
            return {"enabled": False}

        email_config["enabled"] = True
        return email_config

    def load_images(self) -> Dict[str, Optional[str] | int]:
        """
        Loads images configuration and determines overlay files.
        Searches first in program overlays/, then in ~/.config/photopi/overlays/.
        Returns absolute file paths for overlays.
        """
        preview_file: Optional[str] = None
        final_file: Optional[str] = None

        def find_overlays_in_dir(dir_path: str) -> Tuple[Optional[str], Optional[str]]:
            p, f = None, None
            try:
                for file in os.listdir(dir_path):
                    lower_file = file.lower()
                    if "preview" in lower_file and p is None:
                        p = os.path.join(dir_path, file)
                    elif "final" in lower_file and f is None:
                        f = os.path.join(dir_path, file)
                    if p and f:
                        break
            except FileNotFoundError:
                pass
            return p, f

        # Search programm root overlays first
        program_overlays = os.path.join(self._program_root, "overlays")
        p1, f1 = find_overlays_in_dir(program_overlays)
        preview_file = preview_file or p1
        final_file = final_file or f1

        # Then search user overlays if still missing
        if preview_file is None or final_file is None:
            user_overlays = os.path.join(self._user_config_dir, "overlays")
            p2, f2 = find_overlays_in_dir(user_overlays)
            preview_file = preview_file or p2
            final_file = final_file or f2

        base_image_dir = self.config.get(
            "IMAGES",
            "base_image_dir",
            fallback="~/.local/share/photopi/images"
        )
        base_image_dir = os.path.expanduser(base_image_dir)
        base_image_dir = os.path.abspath(base_image_dir)

        max_image_count = 4
        try:
            max_image_count = self.config.getint("IMAGES", "max_image_count", fallback=max_image_count)
        except ValueError:
            max_image_count = 4

        return {
            "base_image_dir": base_image_dir,
            "max_image_count": max_image_count,
            "preview_overlay": preview_file,
            "final_overlay": final_file,
        }

    def load_server(self) -> Dict[str, Optional[str] | int]:
        if not self.config.getboolean("SERVER", "enabled", fallback=False):
            return {"enabled": False}

        host = self.config.get("SERVER", "host", fallback="*").strip()

        try:
            port = self.config.getint("SERVER", "port", fallback=8080)
        except ValueError:
            port = 8080

        return {
            "enabled": True,
            "host": host,
            "port": port,
        }

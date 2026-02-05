from pathlib import Path
from typing import Optional, Any, Literal

from platformdirs import user_config_dir
from pydantic import BaseModel, Field, field_validator, model_validator


class GeneralConfig(BaseModel):
    """Configuration for general application settings."""
    name: str = "PhotoPi"
    language: Literal["en", "de"] = "en"
    cloud_provider: Optional[Literal["nextcloud"]] = None
    welcome_message: str = ""

    @field_validator("language", mode="before")
    @classmethod
    def normalize_language(cls, v: Any) -> str:
        if isinstance(v, str):
            return v.lower().strip()
        return v


class ImageConfig(BaseModel):
    """Configuration for image processing."""
    base_image_dir: Path = Field(default="~/.local/share/photopi/images")
    max_image_count: int = 4
    file_prefix: str = "PhotoPi"
    preview_overlay: Optional[str] = None
    final_overlay: Optional[str] = None

    @field_validator("base_image_dir", mode="before")
    @classmethod
    def validate_base_image_dir(cls, v: Any) -> Path:
        if v and isinstance(v, str) and v.strip():
            return Path(v).expanduser()

        return Path("~/.local/share/photopi/images").expanduser()

    @model_validator(mode="after")
    def find_missing_overlays(self) -> "ImageConfig":
        """
        Automatic overlay discovery logic.
        If overlays are not explicitly set in the config, this validator searches
        in standard locations (program root and user config dir).
        """
        if self.preview_overlay and self.final_overlay:
            return self

        search_dirs = [
            Path.cwd() / "overlays",
            Path(user_config_dir("photopi")) / "overlays",
        ]

        for directory in search_dirs:
            if not directory.is_dir():
                continue

            for file_path in directory.iterdir():
                if not file_path.is_file():
                    continue

                lower_name = file_path.name.lower()

                if self.preview_overlay is None and "preview" in lower_name:
                    self.preview_overlay = str(file_path.absolute())

                if self.final_overlay is None and "final" in lower_name:
                    self.final_overlay = str(file_path.absolute())

        return self


class EmailConfig(BaseModel):
    """Configuration for email sharing."""
    enabled: bool = False
    smtp_server: str = ""
    smtp_port: int = Field(default=587, ge=1, le=65535)
    smtp_user: str = ""
    smtp_password: str = ""
    sender_email: str = ""
    admin_email: str = ""
    subject: str = ""
    headline: str = ""
    body: str = ""
    footer: str = ""


class NextcloudConfig(BaseModel):
    """Configuration for Nextcloud sharing."""
    url: str = ""
    username: str = ""
    password: str = ""
    folder: str = ""


class AppConfig(BaseModel):
    """Configuration object containing all the app's settings."""
    general: GeneralConfig
    images: ImageConfig
    email: EmailConfig
    nextcloud: NextcloudConfig

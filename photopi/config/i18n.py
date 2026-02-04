import builtins
import json
from importlib import resources
from pathlib import Path


class LanguageManager:
    def __init__(self, language: str):
        self.language = language

        try:
            self.lang_dir = Path(str(resources.files("photopi").joinpath("lang")))
        except (ImportError, TypeError):
            self.lang_dir = Path(__file__).parent / "lang"

    def setup(self) -> None:
        """
        Loads the translation files and installs a global _() function.
        """
        primary_data = self._read_json(self.language)
        fallback_data = self._read_json("en")

        translations = {**fallback_data, **primary_data}

        builtins._ = lambda key: translations.get(key, key)

    def get_keyboard_file(self) -> Path:
        """
        Returns the absolute path to the keyboard layout JSON file.
        """
        file_path = self.lang_dir / f"keyboard_{self.language}.json"
        if file_path.is_file():
            return file_path.absolute()

        fallback_path = self.lang_dir / "keyboard_en.json"
        if fallback_path.is_file():
            return fallback_path.absolute()

        raise FileNotFoundError(f"No keyboard layout found in {self.lang_dir}")

    def _read_json(self, code: str) -> dict:
        file_path = self.lang_dir / f"{code}.json"
        if file_path.exists():
            try:
                with open(file_path, encoding="utf-8") as f:
                    return json.load(f)
            except Exception as e:
                print(f"Error loading language {code}: {e}")
        return {}

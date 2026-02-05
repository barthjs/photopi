from abc import ABC, abstractmethod
from typing import Optional

from pydantic import BaseModel


class CloudProvider(ABC):
    """
    Abstract base class acting as a contract for all cloud storage providers.
    """

    @abstractmethod
    def __init__(self, config: BaseModel):
        """
        Initialize the provider with the application configuration.
        This is where credentials from config.ini should be loaded.
        """
        self.config = config

    @abstractmethod
    def upload_files(self, folder_path: str, prefix: Optional[str] = None) -> Optional[str]:
        """
        Uploads all images from the given folder to the cloud service.

        :param folder_path: The local directory containing the images to upload.
        :param prefix: Optional parent folder name on the cloud service.
        :return: A string containing the public share URL, or None if the upload fails.
        """
        pass

    @abstractmethod
    def validate_connection(self) -> bool:
        """
        Checks if the provider can reach the service with the provided credentials.

        :return: True if the connection is successful, False otherwise.
        """
        pass

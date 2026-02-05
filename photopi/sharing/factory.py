from typing import Optional

from photopi.config import AppConfig
from photopi.sharing.base import CloudProvider
from photopi.sharing.nextcloud import NextcloudProvider


class SharingFactory:
    """
    Class responsible for instantiating sharing services based on configuration.
    """

    @staticmethod
    def get_cloud_provider(config: AppConfig) -> Optional[CloudProvider]:
        """
        Resolves and returns the configured cloud provider instance.
        Returns None if no valid provider is configured.
        """

        if config.general.cloud_provider == "nextcloud":
            return NextcloudProvider(config.nextcloud)

        return None

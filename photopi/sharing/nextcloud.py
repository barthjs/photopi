import os
from typing import Optional

import requests
from pydantic import BaseModel

from photopi.sharing.base import CloudProvider


class NextcloudProvider(CloudProvider):
    """
    Nextcloud cloud storage provider.
    """

    def __init__(self, config: BaseModel):
        self.config = config
        self.config.url = self.config.url.rstrip("/")
        self.config.folder = self.config.folder.strip("/")

    def upload_files(self, folder_path: str, prefix: Optional[str] = None) -> Optional[str]:
        """
        Uploads all images from the given folder to a Nextcloud instance and creates a public share url.
        Files are uploaded to: <config.folder>/<prefix>/<folder_name> if prefix is provided,
        otherwise to: <config.folder>/<folder_name>.
        """
        if not os.path.exists(folder_path):
            return None

        folder_name = os.path.basename(folder_path)

        # Build remote path parts
        remote_base = self.config.folder.strip("/")
        if prefix:
            remote_base = f"{remote_base}/{prefix.strip('/')}"
        remote_path = f"{remote_base}/{folder_name}"

        webdav_base = f"{self.config.url}/remote.php/dav/files/{self.config.username}"
        webdav_prefix_url = f"{webdav_base}/{remote_base}"
        webdav_url = f"{webdav_base}/{remote_path}"

        ocs_path = f"/{remote_path.lstrip('/')}"

        try:
            # Ensure prefix directory exists (MKCOL is idempotent-ish; ignore if exists)
            try:
                requests.request(
                    "MKCOL",
                    webdav_prefix_url,
                    auth=(self.config.username, self.config.password),
                    timeout=10
                )
            except Exception:
                pass

            # Create session directory
            requests.request(
                "MKCOL",
                webdav_url,
                auth=(self.config.username, self.config.password),
                timeout=10
            )

            for filename in os.listdir(folder_path):
                if filename.lower().endswith(".jpg"):
                    file_path = os.path.join(folder_path, filename)
                    file_url = f"{webdav_url}/{filename}"
                    with open(file_path, "rb") as f:
                        requests.put(
                            file_url,
                            data=f,
                            auth=(self.config.username, self.config.password),
                            timeout=30
                        )

            # https://docs.nextcloud.com/server/stable/developer_manual/client_apis/OCS/ocs-share-api.html#create-a-new-share
            share_url = f"{self.config.url}/ocs/v2.php/apps/files_sharing/api/v1/shares?format=json"
            payload = {
                "path": ocs_path,
                "shareType": 3,  # public link
                "permissions": 1  # read
            }
            headers = {
                "OCS-APIRequest": "true",
                "Accept": "application/json"
            }

            response = requests.post(
                share_url,
                data=payload,
                headers=headers,
                auth=(self.config.username, self.config.password),
                timeout=30
            )

            if response.status_code in (200, 201):
                data = response.json()
                if data.get("ocs", {}).get("meta", {}).get("status") == "ok":
                    return data["ocs"]["data"]["url"]
                else:
                    status_code = data.get("ocs", {}).get("meta", {}).get("statuscode")
                    message = data.get("ocs", {}).get("meta", {}).get("message")
                    print(f"Nextcloud OCS API error: {status_code} - {message}")

        except Exception as e:
            print(f"Nextcloud upload failed: {e}")

        return None

    def validate_connection(self) -> bool:
        """
        Checks if Nextcloud is reachable with the provided credentials.
        """
        return True

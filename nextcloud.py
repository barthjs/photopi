from photopi.config.config_loader import ConfigLoader
from photopi.sharing.nextcloud import NextcloudProvider


def main():
    images_folder = "./images"

    config_loader = ConfigLoader()
    app_config = config_loader.load_config()

    provider = NextcloudProvider(app_config.nextcloud)

    share_link = provider.upload_files(images_folder)

    if share_link:
        print(f"Upload successful!")
        print(f"Share Link: {share_link}")
    else:
        print("Upload failed. Check your Nextcloud credentials and permissions.")


if __name__ == "__main__":
    main()

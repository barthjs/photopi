import builtins
import os
from importlib import resources

from flask import Flask, render_template, abort

from photopi.config.config_loader import ConfigLoader


def create_app(config_loader: ConfigLoader) -> Flask:
    """
    Create and configure the Flask app
    """
    templates_dir = str(resources.files('photopi').joinpath('templates'))
    base_image_dir = config_loader.config.get(
        "IMAGES",
        "base_image_dir",
        fallback="~/.local/share/photopi/images"
    )
    base_image_dir = os.path.expanduser(base_image_dir)
    base_image_dir = os.path.abspath(base_image_dir)

    app = Flask(__name__, template_folder=templates_dir, static_folder=base_image_dir)
    app.config['base_image_dir'] = base_image_dir

    @app.route('/')
    def home():
        return render_template('home.html')

    @app.route('/images/<directory>')
    def show_gallery(directory):
        gallery_path = os.path.join(app.config['base_image_dir'], directory)
        if not os.path.exists(gallery_path):
            abort(404)

        images = [f for f in os.listdir(gallery_path) if f.endswith('.jpg')]
        return render_template('gallery.html', directory=f'{directory}/', images=images)

    return app

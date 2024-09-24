from flask import Flask, render_template, request, abort, jsonify
import os
from werkzeug.utils import secure_filename
from dotenv import load_dotenv

load_dotenv()
app = Flask(__name__)
IMAGE_DIRECTORY = os.path.join(os.getenv('IMAGES_DIR'))


@app.route('/')
def home():
    return render_template('home.html')


@app.route('/images/<directory>')
def show_gallery(directory):
    gallery_path = os.path.join(IMAGE_DIRECTORY, directory)
    if not os.path.exists(gallery_path):
        abort(404, 'Gallery not found.')

    images = [f for f in os.listdir(gallery_path) if f.endswith(('jpg', 'jpeg', 'png'))]

    return render_template('gallery.html', directory=f'images/{directory}/', images=images)


@app.route('/images/<directory>', methods=['POST'])
def upload_images(directory):
    api_key = request.headers.get('x-api-key')
    if api_key != os.getenv('API_KEY'):
        return jsonify({'error': 'Unauthorized'}), 401

    if 'images' not in request.files:
        return jsonify({'error': 'No images provided'}), 400

    images = request.files.getlist('images')
    saved_files = []
    skipped_files = []

    gallery_path = os.path.join(IMAGE_DIRECTORY, directory)
    if not os.path.exists(gallery_path):
        os.makedirs(gallery_path)

    for image in images:
        filename = secure_filename(image.filename)
        file_path = os.path.join(gallery_path, filename)

        if os.path.exists(file_path):
            skipped_files.append(filename)
        else:
            image.save(file_path)
            saved_files.append(filename)

    return jsonify({
        'message': 'Image upload process completed!',
        'saved_files': saved_files,
        'skipped_files': skipped_files
    }), 201


if __name__ == '__main__':
    app.run(debug=os.getenv('DEBUG', False))

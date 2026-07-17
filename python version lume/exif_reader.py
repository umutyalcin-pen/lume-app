import os
import hashlib
from datetime import datetime
import piexif
from logger_config import logger

SUPPORTED_EXTENSIONS = {'.jpg', '.jpeg', '.tiff', '.tif', '.webp', '.png', '.heic', '.mov', '.mp4'}
MAX_METADATA_FILE_SIZE = 250 * 1024 * 1024
HASH_BUFFER_SIZE = 64 * 1024
QUICK_HASH_READ_SIZE = 4096

SOURCE_PATTERNS = {
    'WhatsApp': ['whatsapp', '-wa', '_wa'],
    'Telegram': ['telegram'],
    'Screenshots': ['screenshot', 'ekran görüntüsü', 'ekran goruntusu', 'screen', 'ekran_goruntusu'],
    'Instagram': ['instagram', 'ig_'],
    'Twitter': ['twitter', 'tw_'],
    'Facebook': ['facebook', 'fb_'],
    'Snapchat': ['snapchat', 'snap-'],
    'Wallpapers': ['wallpaper', 'duvar kağıdı', 'duvar kagidi', 'arkaplan', 'duvar_kagidi'],
    'Downloads': ['download', 'indir'],
    'Gemini AI': ['gemini', 'google_ai'],
    'AI Generated': ['dalle', 'midjourney', 'stable_diffusion']
}

def get_file_hash(file_path: str, quick: bool = True) -> str:

    try:

        real_path = os.path.realpath(file_path)
        abs_path = os.path.abspath(file_path)
        if os.path.normcase(real_path) != os.path.normcase(abs_path):
            logger.warning(f"Symlink detected in hash calculation: {os.path.basename(file_path)}")
            return ""

        stat = os.stat(file_path)

        if quick:

            hasher = hashlib.md5()

            try:
                with open(file_path, 'rb') as f:
                    header = f.read(QUICK_HASH_READ_SIZE)
                    hasher.update(header)
                    header_hash = hasher.hexdigest()[:8]
            except Exception:
                header_hash = "00000000"

            return f"{stat.st_size}_{int(stat.st_mtime)}_{header_hash}"
        else:

            hasher = hashlib.md5()

            with open(file_path, 'rb') as f:
                while True:
                    chunk = f.read(HASH_BUFFER_SIZE)
                    if not chunk:
                        break
                    hasher.update(chunk)

            return hasher.hexdigest()

    except PermissionError:
        logger.error(f"Permission denied: {os.path.basename(file_path)}")
        return ""
    except FileNotFoundError:
        logger.error(f"File not found: {os.path.basename(file_path)}")
        return ""
    except Exception as e:
        logger.warning(f"Hash calculation failed: {os.path.basename(file_path)} - {str(e)}")
        return ""

def get_exif_data(file_path: str) -> dict:

    result = {
        'date': None,
        'date_str': 'No Date Info',
        'device': 'Unknown Device',
        'year': 'Unknown',
        'month': 'Unknown'
    }

    try:

        with open(file_path, 'rb') as f:
            file_bytes = f.read()
        exif_dict = piexif.load(file_bytes)

        if "0th" in exif_dict:
            model = exif_dict["0th"].get(piexif.ImageIFD.Model)
            if model:
                if isinstance(model, bytes):

                    try:
                        model = model.decode('utf-8').strip('\x00').strip()
                    except UnicodeDecodeError:

                        model = model.decode('latin-1', errors='replace').strip('\x00').strip()

                result['device'] = str(model)

        if "Exif" in exif_dict:
            date_raw = exif_dict["Exif"].get(piexif.ExifIFD.DateTimeOriginal)
            if not date_raw:
                date_raw = exif_dict["Exif"].get(piexif.ExifIFD.DateTimeDigitized)

            if date_raw:
                if isinstance(date_raw, bytes):
                    date_str_decoded = date_raw.decode('utf-8', errors='ignore').strip('\x00').strip()
                else:
                    date_str_decoded = str(date_raw)

                try:
                    parsed_date = datetime.strptime(date_str_decoded, "%Y:%m:%d %H:%M:%S")
                    result['date'] = parsed_date
                    result['date_str'] = parsed_date.strftime("%Y-%m-%d %H:%M")
                    result['year'] = str(parsed_date.year)
                    result['month'] = f"{parsed_date.month:02d}"
                except (ValueError, KeyError) as e:
                    logger.debug(f"EXIF date parsing failed: {os.path.basename(file_path)}")

    except Exception as e:
        logger.debug(f"EXIF loading failed: {os.path.basename(file_path)}")

    if result['date'] is None:
        try:
            file_stat = os.stat(file_path)
            creation_time = file_stat.st_ctime
            file_date = datetime.fromtimestamp(creation_time)

            result['date'] = file_date
            result['date_str'] = file_date.strftime("%Y-%m-%d %H:%M") + " (File)"
            result['year'] = str(file_date.year)
            result['month'] = f"{file_date.month:02d}"
        except Exception as e:
            logger.warning(f"File date reading failed: {os.path.basename(file_path)}")

    return result

def is_supported_image(file_path: str) -> bool:

    try:
        ext = os.path.splitext(file_path)[1].lower()
        return ext in SUPPORTED_EXTENSIONS
    except Exception:
        return False

def detect_source(filename: str) -> str:

    try:
        filename_lower = filename.lower()

        for source, patterns in SOURCE_PATTERNS.items():
            for pattern in patterns:
                if pattern in filename_lower:
                    return source

        if filename_lower.startswith(('img_', 'photo_', 'dsc')):
            return "Camera"

        if filename_lower.startswith('20') and len(filename_lower) > 8 and filename_lower[0:4].isdigit():
            return "Camera"

        return None
    except Exception:
        return None

def get_file_info(file_path: str) -> dict:

    try:

        real_path = os.path.realpath(file_path)
        abs_path = os.path.abspath(file_path)
        if os.path.normcase(real_path) != os.path.normcase(abs_path):
            logger.warning(f"Security: Symlink blocked - {os.path.basename(file_path)}")
            return {}

        if file_path.lower().endswith('.lnk'):
            logger.warning(f"Security: Shortcut blocked - {os.path.basename(file_path)}")
            return {}

        if '..' in file_path.split(os.sep):
            logger.warning(f"Security: Path traversal blocked - {os.path.basename(file_path)}")
            return {}

        file_size = os.path.getsize(file_path)
        ext = os.path.splitext(file_path)[1].lower()

        has_exif_support = ext in {'.jpg', '.jpeg', '.tiff', '.tif'}
        if has_exif_support and file_size <= MAX_METADATA_FILE_SIZE:
            exif_data = get_exif_data(file_path)
        else:
            if file_size > MAX_METADATA_FILE_SIZE:
                logger.info(f"Large file ({file_size/1e6:.1f}MB), metadata skipped")

            exif_data = {
                'device': 'Unknown Device',
                'date': None,
                'date_str': 'No Date Info',
                'year': 'Unknown',
                'month': 'Unknown'
            }

            try:
                file_stat = os.stat(file_path)
                file_date = datetime.fromtimestamp(file_stat.st_ctime)
                exif_data.update({
                    'date': file_date,
                    'date_str': file_date.strftime("%Y-%m-%d %H:%M") + " (File)",
                    'year': str(file_date.year),
                    'month': f"{file_date.month:02d}"
                })
            except Exception:
                pass

        filename = os.path.basename(file_path)
        source = detect_source(filename)

        if exif_data['device'] != 'Unknown Device':
            device = exif_data['device']
        else:
            base_name = os.path.splitext(filename)[0]
            parts = base_name.replace('-', '_').replace(' ', '_').split('_')
            if len(parts) > 1 and len(parts[0]) > 2:
                device = parts[0].capitalize()
            else:
                device = "Other"

        return {
            'path': file_path,
            'filename': filename,
            'date': exif_data['date'],
            'date_str': exif_data['date_str'],
            'device': device,
            'source': source,
            'year': exif_data['year'],
            'month': exif_data['month']
        }

    except PermissionError:
        logger.error(f"Permission denied: {os.path.basename(file_path)}")
        return {}
    except Exception as e:
        logger.error(f"Error reading file info: {os.path.basename(file_path)} - {str(e)}")
        return {}

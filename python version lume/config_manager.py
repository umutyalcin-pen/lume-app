"""
Configuration Manager
Saves and reads application settings to a JSON file in APPDATA.
Security Hardened Version
"""

import json
import os

# Security: Whitelist valid configuration keys
VALID_KEYS = {"target_folder", "appearance_mode", "language"}
VALID_MODES = {"dark", "light"}
VALID_LANGS = {"en", "tr"}

def get_config_path():
    """
    Returns the user-specific AppData path for the config file.
    """
    app_data = os.environ.get('APPDATA') or os.path.expanduser('~')
    lume_dir = os.path.join(app_data, 'Lume')
    
    # Create directory if it doesn't exist
    try:
        if not os.path.exists(lume_dir):
            os.makedirs(lume_dir, exist_ok=True)
    except OSError as e:
        print(f"Error creating config directory: {e}")
        return None
        
    return os.path.join(lume_dir, 'config.json')

CONFIG_FILE = get_config_path()

DEFAULT_CONFIG = {
    "target_folder": None,
    "appearance_mode": "dark",
    "language": "en"
}

def load_config() -> dict:
    """
    Reads settings from file. Returns defaults if file doesn't exist.
    """
    if not CONFIG_FILE or not os.path.exists(CONFIG_FILE):
        return DEFAULT_CONFIG.copy()
    
    try:
        with open(CONFIG_FILE, "r", encoding="utf-8") as f:
            config = json.load(f)
            
            # Security: Validate loaded config
            validated_config = DEFAULT_CONFIG.copy()
            
            for key, value in config.items():
                if key not in VALID_KEYS:
                    continue
                
                # Validate values
                if key == "appearance_mode" and value not in VALID_MODES:
                    continue
                if key == "language" and value not in VALID_LANGS:
                    continue
                if key == "target_folder" and value is not None:
                    if not isinstance(value, str) or len(value) > 260:
                        continue
                
                validated_config[key] = value
            
            return validated_config
            
    except (json.JSONDecodeError, ValueError) as e:
        print(f"Config file corrupted, using defaults: {e}")
        return DEFAULT_CONFIG.copy()
    except Exception as e:
        print(f"Error loading config: {e}")
        return DEFAULT_CONFIG.copy()

def save_config(config_data: dict):
    """
    Saves settings to file with validation.
    """
    if not CONFIG_FILE:
        return False
    
    try:
        # Security: Only save validated keys
        safe_config = {}
        for key, value in config_data.items():
            if key in VALID_KEYS:
                safe_config[key] = value
        
        with open(CONFIG_FILE, "w", encoding="utf-8") as f:
            json.dump(safe_config, f, indent=4, ensure_ascii=False)
        return True
        
    except OSError as e:
        print(f"Error saving config (permission denied): {e}")
        return False
    except Exception as e:
        print(f"Error saving config: {e}")
        return False

def update_setting(key: str, value):
    """Updates a single setting in the config file with validation."""
    # Security: Strict validation
    if key not in VALID_KEYS:
        return False

    if key == "appearance_mode" and value not in VALID_MODES:
        return False
        
    if key == "language" and value not in VALID_LANGS:
        return False

    # Validation for target_folder
    if key == "target_folder" and value:
        if not isinstance(value, str):
            return False
        
        # Path length check
        if len(value) > 260:
            return False
        
        # Existence check
        if not os.path.exists(value):
            return False
        
        # Write permission check
        if not os.access(value, os.W_OK):
            return False
        
        # Security: Resolve real path
        try:
            real_path = os.path.realpath(value)
            
            # Block system directories
            system_paths = [
                os.environ.get('SYSTEMROOT', 'C:\\Windows').lower(),
                os.environ.get('PROGRAMFILES', 'C:\\Program Files').lower(),
                os.environ.get('PROGRAMFILES(X86)', 'C:\\Program Files (x86)').lower(),
            ]
            
            real_path_lower = real_path.lower()
            for sys_path in system_paths:
                if real_path_lower.startswith(sys_path):
                    return False
        except Exception:
            return False

    config = load_config()
    config[key] = value
    return save_config(config)

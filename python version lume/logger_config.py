"""
Centralized Logger Configuration
Security Hardened Version
"""

import logging
import sys
import os
from logging.handlers import RotatingFileHandler

def get_log_path():
    """Returns the user-specific AppData path for the log file."""
    app_data = os.environ.get('APPDATA') or os.path.expanduser('~')
    lume_dir = os.path.join(app_data, 'Lume')
    
    try:
        if not os.path.exists(lume_dir):
            os.makedirs(lume_dir, exist_ok=True)
    except OSError:
        return None
    
    return os.path.join(lume_dir, 'app.log')

def setup_logger():
    """Configures the logger with privacy-aware formatting."""
    logger = logging.getLogger("Lume")
    logger.setLevel(logging.INFO)
    
    # Security: Privacy-aware format (no sensitive paths)
    formatter = logging.Formatter(
        '%(asctime)s - %(levelname)s - %(message)s', 
        datefmt='%Y-%m-%d %H:%M:%S'
    )
    
    # Get log path
    log_path = get_log_path()
    
    # File Handler (Max 10MB, 5 backups)
    if log_path:
        try:
            file_handler = RotatingFileHandler(
                log_path, 
                maxBytes=10*1024*1024, 
                backupCount=5, 
                encoding="utf-8"
            )
            file_handler.setFormatter(formatter)
            logger.addHandler(file_handler)
        except Exception:
            print("Warning: Could not initialize log file")
    
    # Console Handler
    console_handler = logging.StreamHandler(sys.stdout)
    console_handler.setFormatter(formatter)
    logger.addHandler(console_handler)
    
    return logger

# Export the main logger
logger = setup_logger()

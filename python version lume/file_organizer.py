"""
File Organization Module
Security Hardened Version - Copy-Verify-Delete Method
"""

import os
import shutil
import filecmp
from typing import Dict, Tuple
from logger_config import logger
from exif_reader import get_file_hash

# Security Constants
MAX_CONFLICT_WARNING = 1000
MAX_CONFLICT_LIMIT = 10000

def sanitize_folder_name(name: str) -> str:
    """
    Sanitizes folder name for filesystem safety.
    Blocks: path traversal, invalid chars, reserved names.
    """
    # Security 1: Block path traversal
    if name in (".", ".."):
        return "Unknown"
    
    # Security 2: Remove invalid characters
    invalid_chars = '<>:"/\\|?*.'
    for char in invalid_chars:
        name = name.replace(char, '_')
    
    # Security 3: Remove control characters (0-31)
    name = "".join(c if ord(c) > 31 else "_" for c in name)
    
    # Security 4: Strip spaces and dots
    name = name.strip().strip('.')
    
    # Security 5: Windows reserved names
    reserved = {
        "CON", "PRN", "AUX", "NUL",
        "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
        "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"
    }
    if name.upper() in reserved:
        name = f"{name}_safe"
    
    # Security 6: Length limit
    if len(name) > 100:
        name = name[:100]
    
    # Default if empty
    if not name:
        name = "Unknown"
    
    return name

def generate_target_path(base_path: str, year: str, month: str, device: str, filename: str) -> str:
    """
    Constructs the target file path.
    Structure: Target_Folder/YEAR/MONTH/DEVICE/file.ext
    """
    safe_year = sanitize_folder_name(year)
    safe_month = sanitize_folder_name(month)
    safe_device = sanitize_folder_name(device)
    
    target_dir = os.path.join(base_path, safe_year, safe_month, safe_device)
    
    return os.path.join(target_dir, filename)

def handle_conflict(source_path: str, target_path: str) -> Tuple[str, bool]:
    """
    Handles naming conflicts and checks for duplicates.
    Returns: (final_path, is_duplicate)
    """
    # Security: Use lexists to handle broken symlinks
    if not os.path.lexists(target_path):
        return target_path, False
    
    # Performance 1: Size comparison first
    try:
        source_size = os.path.getsize(source_path)
        target_size = os.path.getsize(target_path)
        
        if source_size != target_size:
            # Different sizes = definitely different files
            # Generate unique name
            base, ext = os.path.splitext(target_path)
            counter = 1
            while os.path.lexists(f"{base}_{counter}{ext}"):
                counter += 1
                if counter > MAX_CONFLICT_LIMIT:
                    raise Exception(f"Conflict limit exceeded: {os.path.basename(source_path)}")
            return f"{base}_{counter}{ext}", False
    except OSError as e:
        logger.warning(f"Size comparison failed: {e}")
    
    # Performance 2: Deep content comparison
    try:
        if filecmp.cmp(source_path, target_path, shallow=False):
            logger.info(f"Duplicate detected: {os.path.basename(source_path)}")
            return target_path, True
    except PermissionError:
        logger.error(f"Permission denied comparing: {os.path.basename(source_path)}")
    except Exception as e:
        logger.warning(f"Comparison failed: {os.path.basename(source_path)} - {e}")
    
    # Not identical - generate unique name
    base, ext = os.path.splitext(target_path)
    counter = 1
    
    while True:
        new_path = f"{base}_{counter}{ext}"
        
        if not os.path.lexists(new_path):
            return new_path, False
        
        # Check if identical to numbered file
        try:
            if filecmp.cmp(source_path, new_path, shallow=False):
                logger.info(f"Duplicate found at {counter}: {os.path.basename(source_path)}")
                return new_path, True
        except Exception:
            pass
        
        # Logging for high conflicts
        if counter == MAX_CONFLICT_WARNING:
            logger.warning(f"High conflict count ({counter}) for: {os.path.basename(source_path)}")
        
        counter += 1
        
        # Security: Prevent infinite loop
        if counter > MAX_CONFLICT_LIMIT:
            logger.error(f"Conflict limit exceeded ({MAX_CONFLICT_LIMIT})")
            raise Exception(f"Too many conflicts for: {os.path.basename(source_path)}")

def ensure_directory(path: str) -> None:
    """Ensures directory exists with error handling."""
    try:
        os.makedirs(path, exist_ok=True)
    except OSError as e:
        logger.error(f"Failed to create directory: {e}")
        raise

def move_file(file_info: Dict, target_base: str) -> bool:
    """
    Moves file safely using copy-verify-delete method.
    Returns: True if successful or duplicate
    """
    source = None
    final_target = None
    
    try:
        source = file_info['path']
        filename = os.path.basename(source)
        
        # Security: Validate source exists
        if not os.path.exists(source):
            logger.error(f"Source file not found: {filename}")
            return False
        
        target = calculate_new_path(file_info, target_base)
        
        # Security: Validate target path using pathlib
        from pathlib import Path
        target_path = Path(target).resolve()
        base_path = Path(target_base).resolve()
        
        try:
            target_path.relative_to(base_path)
        except ValueError:
            logger.error(f"Security: Invalid target path for {filename}")
            return False
        
        # Handle conflict & duplicate check
        final_target, is_duplicate = handle_conflict(source, target)
        
        if is_duplicate:
            rel_path = os.path.relpath(final_target, target_base)
            logger.info(f"Skipping duplicate: {filename} (exists at {rel_path})")
            return True
        
        # Create target directory
        target_dir = os.path.dirname(final_target)
        ensure_directory(target_dir)
        
        # Pre-copy integrity check
        source_hash = file_info.get('quick_hash') or get_file_hash(source, quick=False)
        
        # SAFE METHOD: Copy -> Verify -> Delete (instead of move)
        # Step 1: Copy file to target
        shutil.copy2(source, final_target)
        
        # Step 2: Verify copy integrity
        if source_hash:
            target_hash = get_file_hash(final_target, quick=False)
            
            if target_hash != source_hash:
                logger.error(f"Integrity check FAILED for {filename}! Removing corrupt copy...")
                
                # Remove corrupt copy
                try:
                    os.remove(final_target)
                    logger.info(f"Corrupt copy removed, source preserved: {filename}")
                except Exception as remove_err:
                    logger.critical(f"CRITICAL: Failed to remove corrupt copy: {remove_err}")
                return False
        
        # Step 3: Delete source only after verified copy
        try:
            os.remove(source)
        except PermissionError:
            logger.warning(f"Could not delete source (permission denied): {filename}")
            # Copy succeeded, source still exists - acceptable state
        except Exception as del_err:
            logger.warning(f"Could not delete source: {filename} - {del_err}")
            # Copy succeeded, source still exists - acceptable state
        
        rel_path = os.path.relpath(final_target, target_base)
        logger.info(f"Archived: {filename} -> {rel_path}")
        return True
    
    except KeyError as e:
        logger.error(f"Missing file info key: {e}")
        return False
    except PermissionError:
        if source:
            logger.error(f"Permission denied: {os.path.basename(source)}")
        return False
    except OSError as e:
        if source:
            logger.error(f"System error: {os.path.basename(source)} - {e}")
        return False
    except Exception as e:
        if source:
            logger.error(f"Unexpected error: {os.path.basename(source)} - {str(e)}")
        else:
            logger.error(f"Unexpected error in move_file: {str(e)}")
        return False

def calculate_new_path(file_info: dict, target_base: str) -> str:
    """Calculates new path based on file info."""
    source = file_info.get('source')
    device = file_info.get('device', 'Unknown Device')
    
    if source:
        source_device = f"{source}_{device}"
    else:
        source_device = device
    
    return generate_target_path(
        base_path=target_base,
        year=file_info['year'],
        month=file_info['month'],
        device=source_device,
        filename=file_info['filename']
    )

def get_relative_path(full_path: str, base_path: str) -> str:
    """Returns relative path from base_path."""
    try:
        return os.path.relpath(full_path, base_path)
    except ValueError:
        return full_path

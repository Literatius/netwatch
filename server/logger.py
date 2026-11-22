import logging

logger = logging.getLogger("netwatch")

logger.setLevel(logging.INFO)
file_handle = logging.FileHandler("monitor.log", encoding="utf-8")

logger.addHandler(file_handle)

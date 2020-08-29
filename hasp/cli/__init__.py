import logging.config
import os
from typing import Dict
from functools import partial

import click
import pkg_resources
from rich.console import Console
from rich.traceback import install as rich_traceback_install
from ruamel.yaml import YAML


TRUTHY = ["1", "t", "true", "y", "yes"]

option_ = partial(click.option, show_default=True, show_envvar=True)


def not_implemented():
    raise click.UsageError("Sorry, this command is known but not implemented yet. Come back later!")


def shim_env_vars(config_data: Dict):
    """Mutates logging configuration dictionary in-place before it is loaded ie. early.

    Args:
        config_data: Configuration dictionary destined for logging.config.dictConfig().

    Environment Variables:
        HASP_DEBUG:           STDOUT debug level
        HASP_LOG_ENABLE_AWS:  Allow boto3 modules to propagate logging messages
        HASP_LOG_FILE:        Log file location
        HASP_DEV:             Use "rich" handler

    """

    # Force DEBUG level logging
    if "HASP_DEBUG" in os.environ.keys():
        if os.environ["HASP_DEBUG"].lower() in TRUTHY:
            config_data["handlers"]["console"]["level"] = "DEBUG"

    # Enable verbose boto3 logging
    if "HASP_LOG_ENABLE_AWS" in os.environ.keys():
        if os.environ["HASP_LOG_ENABLE_AWS"].lower() in TRUTHY:
            for logger_ in ["boto3", "botocore", "urllib3"]:
                config_data["loggers"][logger_]["propagate"] = True

    # Custom log path
    if "HASP_LOG_FILE" in os.environ.keys():
        log_file = os.environ["HASP_LOG_FILE"]
        log_file_name = log_file.split(".")[0:-1][0]
        log_file_suffix = log_file.split(".")[-1]

        config_data["handlers"]["file"]["filename"] = log_file
        config_data["handlers"]["file-debug"]["filename"] = f"{log_file_name}-debug.{log_file_suffix}"

    # Use 'rich.logging.RichHandler' with stdout (instead of 'logging.StreamHandler')
    if "HASP_DEV" in os.environ.keys():
        if os.environ["HASP_DEV"].lower() in TRUTHY:
            config_data["root"]["handlers"] = ["rich"]
            config_data["handlers"]["rich"]["level"] = "DEBUG"


def read_config():
    yaml = YAML(typ="safe", pure=True)
    data = yaml.load(pkg_resources.resource_string("hasp.resources", "log_conf.yaml"))
    return data


CONFIG_DATA: Dict = read_config()
shim_env_vars(CONFIG_DATA)
logging.config.dictConfig(CONFIG_DATA)

console = Console(log_time=False)
rich_traceback_install(console=console)

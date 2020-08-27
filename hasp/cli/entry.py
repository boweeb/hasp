"""Console script for hasp."""
import logging
import sys
from functools import partial
from importlib.metadata import version
from subprocess import run
from typing import List, Text

import click
from rich import print as rprint

from hasp.cli import console

from hasp.cli import verb as v


LOG = logging.getLogger(__name__)
LOG.debug("[entry.py]")

# Add defaults
option_ = partial(click.option, show_default=True, show_envvar=True)


# CLI ROOT
@click.group(context_settings=dict(max_content_width=120))
@option_("--verbose", "-v", is_flag=True, required=False, help="Enable debug-level logging")
@option_("--config-file", "-c", required=False, help="Path to configuration file")
@option_("--state-file", "-s", required=False, help="Path to state file")
@option_("--ssh-dir", "-w", envvar="WORKSPACE", help="Path to common home for projects")
@click.version_option(version=version("hasp"))
@click.pass_context
def root(ctx, verbose, config_file, state_file, ssh_dir):
    """hasp -- Make sense of your SSH"""
    LOG.debug("[ROOT]")


def main():
    LOG.debug("[main]")

    root.add_command(v.new)
    root.add_command(v.edit)
    root.add_command(v.find)
    root.add_command(v.list_)
    root.add_command(v.show)

    root(
        auto_envvar_prefix="HASP",
        # https://github.com/pallets/click/pull/1566
        # When this gets merged (and released) show_default will propagate to sub-commands
        show_default=True,
    )


if __name__ == "__main__":
    sys.exit(main())  # pragma: no cover

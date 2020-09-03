"""Console script for hasp."""

import logging
import sys
from functools import partial
from importlib.metadata import version
from subprocess import run
from typing import List, Text
from pathlib import Path

import click
from rich import print as rprint

from hasp.cli import console

from hasp.cli import verb as v
from hasp.io import ContextBorg
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from hasp.orm import Base


LOG = logging.getLogger(__name__)
LOG.debug("[entry.py]")

# Add defaults
option_ = partial(click.option, show_default=True, show_envvar=True)


# CLI ROOT
@click.group(context_settings=dict(max_content_width=120))
@option_("--verbose", "-v", is_flag=True, required=False, help="Enable debug-level logging")
@option_("--config-file", "-c", default="~/.config/hasp/config.toml", required=False, help="Path to configuration file")
@option_("--state-file", "-s", default="~/.local/share/hasp/state.toml", required=False, help="Path to state file")
@option_("--ssh-dir", "-w", default="~/.ssh", help="Path to common home for projects")
@click.version_option(version=version("hasp"))
@click.pass_context
def root(ctx, verbose, config_file, state_file, ssh_dir):
    """hasp -- Make sense of your SSH"""
    LOG.debug("[ROOT]")
    ctx.ensure_object(dict)
    ctx.obj["config_file"] = Path(config_file).expanduser()
    ctx.obj["state_file"] = Path(state_file).expanduser()
    ctx.obj["ssh_dir"] = Path(ssh_dir).expanduser()

    db_path = "/tmp/quicktest.db"
    db_opts = {
        "check_same_thread": True,
        "timeout": 10,
        "uri": True,
    }
    db_opts_str = "&".join(["=".join([key, str(value).lower()]) for key, value in db_opts.items()])
    engine = create_engine(
        f'sqlite:///file:{db_path}?{db_opts_str}',
        # echo=True,
    )
    _session = sessionmaker(bind=engine)
    session = _session()

    Base.metadata.create_all(engine)

    _ = ContextBorg(init_data={**ctx.obj, "session": session})


def main():
    LOG.debug("[main]")

    for group in [v.new, v.edit, v.find, v.list_, v.show, v.sync]:
        root.add_command(group)

    root(
        auto_envvar_prefix="HASP",
        # https://github.com/pallets/click/pull/1566
        # When this gets merged (and released) show_default will propagate to sub-commands
        show_default=True,
    )


if __name__ == "__main__":
    sys.exit(main())  # pragma: no cover

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


LOG = logging.getLogger(__name__)

# Add defaults
option_ = partial(click.option, show_default=True, show_envvar=True)


# CLI ROOT
@click.group(context_settings=dict(max_content_width=120))
@option_(
    "--type", "-t", "type_", type=click.Choice(["self-managed", "pyenv", "poetry"]), default="self-managed",
    help="Use alternative function when resource exists"
)
@option_("--verbose", "-v", is_flag=True, required=False, help="Enable debug-level logging")
@option_("--config-file", "-c", required=False, help="Path to configuration file")
@option_("--state-file", "-s", required=False, help="Path to state file")
@option_("--workspace", "-w", envvar="WORKSPACE", help="Path to common home for projects")
@click.version_option(version=version("hasp"))
@click.pass_context
def root(ctx, type_, verbose, config_file, state_file, workspace):
    """hasp -- Make sense of your SSH"""
    sys.exit(0)
    if verbose:
        for handler in logging.getLogger().handlers[:]:
            handler.setLevel("DEBUG")

    config = dict(Config(config_file))
    state = dict(State(state_file))

    workspace = validate_workspace(config, workspace)

    ctx.ensure_object(dict)
    ctx.obj["type_"] = type_
    ctx.obj["verbose"] = verbose
    ctx.obj["config"] = config
    ctx.obj["state"] = state
    ctx.obj["workspace"] = workspace

    # LOG.debug("|| Click context (ctx):")
    # for line in pformat(ctx.obj).split("\n"):
    #     LOG.debug(f"||    {line}")


def main():
    root(
        auto_envvar_prefix="HASP",
        # https://github.com/pallets/click/pull/1566
        # When this gets merged (and released) show_default will propagate to sub-commands
        show_default=True,
    )


if __name__ == "__main__":
    sys.exit(main())  # pragma: no cover

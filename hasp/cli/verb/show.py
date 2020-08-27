"""Compose "show" command subgroup."""

import logging

import click

from hasp.cli.resource.key import show as key
from hasp.cli.resource.host import show as host
from hasp.cli.resource.config import show as config


LOG = logging.getLogger(__name__)
LOG.debug("[show.py]")


# ----------------------------------------------------------------------------------------------------------------------
# NEW
@click.group()
@click.pass_context
def show(
    ctx,
):
    """Report resource details."""
    LOG.debug("[EDIT]")


show.add_command(key)
show.add_command(host)
show.add_command(config)

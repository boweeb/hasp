"""Compose "edit" command subgroup."""

import logging

import click

from hasp.cli.resource.key import edit as key
from hasp.cli.resource.host import edit as host
from hasp.cli.resource.config import edit as config


LOG = logging.getLogger(__name__)
LOG.debug("[edit.py]")


# ----------------------------------------------------------------------------------------------------------------------
# NEW
@click.group()
@click.pass_context
def edit(
    ctx,
):
    """Edit a resource."""
    LOG.debug("[EDIT]")


edit.add_command(key)
edit.add_command(host)
edit.add_command(config)

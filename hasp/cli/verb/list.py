"""Compose "list" command subgroup."""

import logging

import click

from hasp.cli.resource.key import list_ as key
from hasp.cli.resource.host import list_ as host
from hasp.cli.resource.config import list_ as config


LOG = logging.getLogger(__name__)
LOG.debug("[list.py]")


# ----------------------------------------------------------------------------------------------------------------------
# NEW
@click.group("list")
@click.pass_context
def list_(
    ctx,
):
    """List all of a resource type."""
    LOG.debug("[EDIT]")


list_.add_command(key)
list_.add_command(host)
list_.add_command(config)

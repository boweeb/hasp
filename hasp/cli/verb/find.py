"""Compose "find" command subgroup."""

import logging

import click

from hasp.cli.resource.key import find as key
from hasp.cli.resource.host import find as host
from hasp.cli.resource.config import find as config


LOG = logging.getLogger(__name__)
LOG.debug("[find.py]")


# ----------------------------------------------------------------------------------------------------------------------
# NEW
@click.group()
@click.pass_context
def find(
    ctx,
):
    """Search for a resource."""
    LOG.debug("[EDIT]")


find.add_command(key)
find.add_command(host)
find.add_command(config)

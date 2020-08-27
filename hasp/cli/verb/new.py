"""Compose "new" command subgroup."""

import logging

import click

from hasp.cli.resource.key import new as key
from hasp.cli.resource.host import new as host
from hasp.cli.resource.config import new as config


LOG = logging.getLogger(__name__)
LOG.debug("[new.py]")


# ----------------------------------------------------------------------------------------------------------------------
# NEW
@click.group()
@click.pass_context
def new(
    ctx,
):
    """Create new resource."""
    LOG.debug("[NEW]")


new.add_command(key)
new.add_command(host)
new.add_command(config)

"""Compose "find" command subgroup."""

import logging

import click

from hasp.cli.resource.key import find as key
from hasp.cli.resource.host import find as host
from hasp.cli.resource.config import find as config


LOG = logging.getLogger(__name__)
LOG.debug("[find.py]")


@click.group("find")
@click.pass_context
def group(
    ctx,
):
    """Search for a resource."""
    LOG.debug("[FIND]")


group.add_command(key)
group.add_command(host)
group.add_command(config)

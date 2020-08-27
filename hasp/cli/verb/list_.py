"""Compose "list" command subgroup."""

import logging

import click

from hasp.cli.resource.key import list_ as key
from hasp.cli.resource.host import list_ as host
from hasp.cli.resource.config import list_ as config


LOG = logging.getLogger(__name__)
LOG.debug("[list_.py]")


@click.group("list")
@click.pass_context
def group(
    ctx,
):
    """List all of a resource type."""
    LOG.debug("[LIST]")


for sub_group in [key, host, config]:
    group.add_command(sub_group)

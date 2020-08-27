"""Compose "new" command subgroup."""

import logging

import click

from hasp.cli.resource.key import new as key
from hasp.cli.resource.host import new as host
from hasp.cli.resource.config import new as config


LOG = logging.getLogger(__name__)
LOG.debug("[new.py]")


@click.group("new")
@click.pass_context
def group(
    ctx,
):
    """Create new resource."""
    LOG.debug("[NEW]")


for sub_group in [key, host, config]:
    group.add_command(sub_group)

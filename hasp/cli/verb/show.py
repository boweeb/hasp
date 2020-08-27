"""Compose "show" command subgroup."""

import logging

import click

from hasp.cli.resource.key import show as key
from hasp.cli.resource.host import show as host
from hasp.cli.resource.config import show as config


LOG = logging.getLogger(__name__)
LOG.debug("[show.py]")


@click.group("show")
@click.pass_context
def group(
    ctx,
):
    """Report resource details."""
    LOG.debug("[SHOW]")


for sub_group in [key, host, config]:
    group.add_command(sub_group)

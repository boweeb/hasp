"""Compose "sync" command subgroup."""

import logging

import click

from hasp.cli.resource.key import sync as key
from hasp.cli.resource.host import sync as host
from hasp.cli.resource.config import sync as config


LOG = logging.getLogger(__name__)
LOG.debug("[sync.py]")


@click.group("sync")
@click.pass_context
def group(
    ctx,
):
    """Synchronize state files."""
    LOG.debug("[SYNC]")


for sub_group in [key, host, config]:
    group.add_command(sub_group)
